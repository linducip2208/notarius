package app

// Wire workflow engine (migrasi 027) ke permohonan nyata.
//
//   - seedWorkflowTemplates: dipanggil dari dmsChecklistSeed (dms.go) agar
//     satu tombol memasang paket checklist + template alur kerja.
//   - ensureWorkflowInstance: dipanggil dari requestCreate (app.go); gagal
//     diam-diam agar pembuatan permohonan tak pernah digagalkan workflow.
//   - workflowStep: POST /requests/{id}/workflow/step — menandai step
//     current DONE lalu memajukan current_step. SENGAJA tidak menyentuh
//     service_requests.status / requestMove (independen, sesuai tugas).
//
// Keputusan RBAC (disederhanakan, tercatat): langkah ber-required_role
// notaris mensyaratkan PermApproveDeed via rbac.Has; langkah lain cukup
// PermEditClient (sudah dicek di awal = staf ke atas). Nama aktor dicatat
// di workflow_step_states.actor untuk audit.

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

// seedWorkflowTemplates memasang 3 template standar bila belum ada.
// Idempoten per (office, service_code, version=1): template existing
// dilewati (tapi step yang kurang di-backfill), tabel 027 yang hilang
// (DB lama) menghentikan diam-diam. Kembalian = template BARU terpasang.
func (a *App) seedWorkflowTemplates(officeID int64) int {
	made := 0
	for _, s := range standardWorkflowSeeds() {
		var tplID int64
		err := a.db.QueryRow(`SELECT id FROM workflow_templates WHERE office_id=? AND service_code=? AND version=1`,
			officeID, s.code).Scan(&tplID)
		if err == nil {
			a.ensureWorkflowSteps(tplID, s) // backfill best-effort
			continue
		}
		if err != sql.ErrNoRows {
			if isTableMissing(err) {
				return made
			}
			continue
		}
		res, err := a.db.Exec(`INSERT INTO workflow_templates(office_id, service_code, name, version, active)
			VALUES(?,?,?,1,1)`, officeID, s.code, s.name)
		if err != nil {
			if isTableMissing(err) {
				return made
			}
			continue
		}
		tplID, _ = res.LastInsertId()
		if tplID == 0 {
			continue
		}
		if a.ensureWorkflowSteps(tplID, s) {
			made++
		} else {
			made++ // baris template sudah terpasang; step menyusul di klik berikut
		}
	}
	return made
}

// ensureWorkflowSteps memasukkan step template secara idempoten
// (INSERT IGNORE atas uq_wf_step). False bila tabel 027 belum ada.
// SLA 033: sla_days ikut ditulis; baris existing yang masih NULL
// di-backfill agar seed ulang tidak perlu hapus template.
func (a *App) ensureWorkflowSteps(tplID int64, s workflowSeed) bool {
	for i, st := range s.steps {
		var sla any
		if st.slaDays > 0 {
			sla = st.slaDays
		}
		if _, err := a.db.Exec(`INSERT IGNORE INTO workflow_steps(template_id, step_key, label, sort_order, required_role, require_checklist, require_evidence, skippable, sla_days)
			VALUES(?,?,?,?,?,?,?,?,?)`, tplID, st.key, st.label, i+1, st.requiredRole,
			boolToInt(st.requireChecklist), nullStrOpt(st.requireEvidence), boolToInt(st.skippable), sla); err != nil {
			if !isUnknownColumn(err) {
				return false
			}
			// DB lama pra-migrasi 033: tulis tanpa sla_days (fail-open).
			if _, err2 := a.db.Exec(`INSERT IGNORE INTO workflow_steps(template_id, step_key, label, sort_order, required_role, require_checklist, require_evidence, skippable)
				VALUES(?,?,?,?,?,?,?,?)`, tplID, st.key, st.label, i+1, st.requiredRole,
				boolToInt(st.requireChecklist), nullStrOpt(st.requireEvidence), boolToInt(st.skippable)); err2 != nil {
				return false
			}
			continue
		}
		if st.slaDays > 0 {
			// Best-effort: DB yang di-seed sebelum migrasi 033 masih NULL.
			_, _ = a.db.Exec(`UPDATE workflow_steps SET sla_days=? WHERE template_id=? AND step_key=? AND (sla_days IS NULL OR sla_days<>?)`,
				st.slaDays, tplID, st.key, st.slaDays)
		}
	}
	return true
}

// ensureWorkflowInstance membuat instance dari template aktif service_code
// permohonan. Nil bila: tak ada template aktif, instance sudah ada, atau
// tabel 027 hilang (DB lama) — semua dilewati diam-diam.
func (a *App) ensureWorkflowInstance(officeID, requestID int64) error {
	if a.db == nil || officeID <= 0 || requestID <= 0 {
		return nil
	}
	var code sql.NullString
	if err := a.db.QueryRow(`SELECT s.code FROM service_requests r
		JOIN services s ON s.id=r.service_id
		WHERE r.id=? AND r.office_id=?`, requestID, officeID).Scan(&code); err != nil || !code.Valid {
		return nil
	}
	var tplID int64
	if err := a.db.QueryRow(`SELECT id FROM workflow_templates
		WHERE office_id=? AND service_code=? AND active=1
		ORDER BY version DESC LIMIT 1`, officeID, strings.TrimSpace(code.String)).Scan(&tplID); err != nil {
		return nil // ErrNoRows (belum di-seed) maupun tabel hilang: lewati
	}
	var have int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM workflow_instances WHERE request_id=?`, requestID).Scan(&have); err != nil {
		if isTableMissing(err) {
			return nil
		}
		return err
	}
	if have > 0 {
		return nil
	}
	type stepDef struct {
		key string
		sla sql.NullInt64
	}
	var defs []stepDef
	if rows, qerr := a.db.Query(`SELECT step_key, sla_days FROM workflow_steps WHERE template_id=? ORDER BY sort_order, step_key`, tplID); qerr == nil {
		for rows.Next() {
			var d stepDef
			if err := rows.Scan(&d.key, &d.sla); err == nil {
				defs = append(defs, d)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	} else if isTableMissing(qerr) {
		return nil // tabel 027 hilang (DB lama): lewati diam-diam
	} else if isUnknownColumn(qerr) {
		// DB lama pra-migrasi 033: tanpa sla_days (due_at dikosongkan).
		rows, err := a.db.Query(`SELECT step_key FROM workflow_steps WHERE template_id=? ORDER BY sort_order, step_key`, tplID)
		if err != nil {
			if isTableMissing(err) {
				return nil
			}
			return err
		}
		for rows.Next() {
			var d stepDef
			if err := rows.Scan(&d.key); err == nil {
				defs = append(defs, d)
			}
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	} else {
		return qerr
	}
	if len(defs) == 0 {
		return nil // tanpa langkah: jangan bentuk instance setengah jadi
	}
	var keys []string
	for _, d := range defs {
		keys = append(keys, d.key)
	}
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO workflow_instances(office_id, request_id, template_id, current_step, status)
		VALUES(?,?,?,?,'ACTIVE')`, officeID, requestID, tplID, keys[0])
	if err != nil {
		return err
	}
	instID, _ := res.LastInsertId()
	for _, d := range defs {
		var execErr error
		if d.sla.Valid && d.sla.Int64 > 0 {
			// SLA 033: due_at = CURDATE + target hari langkah.
			_, execErr = tx.Exec(`INSERT IGNORE INTO workflow_step_states(instance_id, step_key, status, due_at)
				VALUES(?,?,'PENDING',DATE_ADD(CURDATE(), INTERVAL ? DAY))`, instID, d.key, d.sla.Int64)
			if execErr != nil && isUnknownColumn(execErr) {
				// DB lama pra-033: tulis tanpa due_at (fail-open).
				_, execErr = tx.Exec(`INSERT IGNORE INTO workflow_step_states(instance_id, step_key, status)
					VALUES(?,?,'PENDING')`, instID, d.key)
			}
		} else {
			_, execErr = tx.Exec(`INSERT IGNORE INTO workflow_step_states(instance_id, step_key, status)
				VALUES(?,?,'PENDING')`, instID, d.key)
		}
		if execErr != nil {
			return execErr
		}
	}
	return tx.Commit()
}

// wfStepView adalah satu baris kartu "Alur Kerja" di request_detail.
type wfStepView struct {
	Key, Label, Status, RequiredRole string
	Skippable, Current               bool
	DueAt                            string // 033: jatuh tempo langkah ("" bila tanpa target)
	Overdue                          bool   // 033: due_at < hari ini dan belum DONE
}

// workflowView memuat instance + step terurut (states PENDING default bila
// belum ada baris). has=false bila tak ada instance / tabel 027 hilang.
// SLA 033: due_at dibaca best-effort (DB lama pra-033 → DueAt kosong).
func (a *App) workflowView(officeID, requestID int64) (steps []wfStepView, current, status string, has bool) {
	if a.db == nil {
		return nil, "", "", false
	}
	var instID int64
	var cur sql.NullString
	var st string
	if err := a.db.QueryRow(`SELECT id, current_step, status FROM workflow_instances
		WHERE office_id=? AND request_id=?`, officeID, requestID).Scan(&instID, &cur, &st); err != nil {
		return nil, "", "", false
	}
	rows, err := a.db.Query(`SELECT ws.step_key, ws.label,
		COALESCE(wss.status,'PENDING'), ws.required_role, ws.skippable,
		DATE_FORMAT(wss.due_at,'%Y-%m-%d')
		FROM workflow_steps ws LEFT JOIN workflow_step_states wss
		ON wss.instance_id=? AND wss.step_key=ws.step_key
		WHERE ws.template_id=(SELECT template_id FROM workflow_instances WHERE id=?)
		ORDER BY ws.sort_order, ws.step_key`, instID, instID)
	if err != nil {
		if !isUnknownColumn(err) {
			return nil, "", "", false
		}
		// DB lama pra-033: tanpa due_at.
		return a.workflowViewLegacy(officeID, requestID)
	}
	defer rows.Close()
	today := time.Now().Format("2006-01-02")
	for rows.Next() {
		var v wfStepView
		var skip int
		var due sql.NullString
		if err := rows.Scan(&v.Key, &v.Label, &v.Status, &v.RequiredRole, &skip, &due); err != nil {
			return nil, "", "", false
		}
		v.Skippable = skip == 1
		if due.Valid && strings.TrimSpace(due.String) != "" {
			v.DueAt = strings.TrimSpace(due.String)
			v.Overdue = v.DueAt < today && !strings.EqualFold(strings.TrimSpace(v.Status), "DONE")
		}
		steps = append(steps, v)
	}
	if err := rows.Err(); err != nil {
		return nil, "", "", false
	}
	if cur.Valid {
		current = cur.String
		for i := range steps {
			if steps[i].Key == current {
				steps[i].Current = true
			}
		}
	}
	return steps, current, st, true
}

// workflowViewLegacy memuat kartu Alur Kerja tanpa kolom 033 (DB lama).
func (a *App) workflowViewLegacy(officeID, requestID int64) (steps []wfStepView, current, status string, has bool) {
	var instID int64
	var cur sql.NullString
	var st string
	if err := a.db.QueryRow(`SELECT id, current_step, status FROM workflow_instances
		WHERE office_id=? AND request_id=?`, officeID, requestID).Scan(&instID, &cur, &st); err != nil {
		return nil, "", "", false
	}
	rows, err := a.db.Query(`SELECT ws.step_key, ws.label,
		COALESCE(wss.status,'PENDING'), ws.required_role, ws.skippable
		FROM workflow_steps ws LEFT JOIN workflow_step_states wss
		ON wss.instance_id=? AND wss.step_key=ws.step_key
		WHERE ws.template_id=(SELECT template_id FROM workflow_instances WHERE id=?)
		ORDER BY ws.sort_order, ws.step_key`, instID, instID)
	if err != nil {
		return nil, "", "", false
	}
	defer rows.Close()
	for rows.Next() {
		var v wfStepView
		var skip int
		if err := rows.Scan(&v.Key, &v.Label, &v.Status, &v.RequiredRole, &skip); err != nil {
			return nil, "", "", false
		}
		v.Skippable = skip == 1
		steps = append(steps, v)
	}
	if err := rows.Err(); err != nil {
		return nil, "", "", false
	}
	if cur.Valid {
		current = cur.String
		for i := range steps {
			if steps[i].Key == current {
				steps[i].Current = true
			}
		}
	}
	return steps, current, st, true
}

// wfStepOverdue adalah satu langkah PENDING yang lewat due_at-nya.
type wfStepOverdue struct {
	InstanceID, RequestID int64
	ReqTitle, Service     string
	StepKey, Label        string
	DueAt                 string
	LateDays              int
}

// wfEscalationTitle membangun judul tugas eskalasi SLA langkah.
// Murni tanpa DB agar mudah diuji; SENGAJA stabil (tanpa umur hari) supaya
// dedup 1x via COUNT(title) tetap cocok keesokan harinya.
func wfEscalationTitle(requestID int64, stepKey, label string) string {
	return fmt.Sprintf("Eskalasi SLA langkah %s req #%d (%s)",
		strings.TrimSpace(stepKey), requestID, strings.TrimSpace(label))
}

// workflowStepOverdue memindai langkah PENDING yang due_at-nya lewat
// (due_at < CURDATE). Fail-open: nil bila DB nil / tabel 027 hilang /
// kolom 033 belum ada — widget dashboard menyembunyikan diri.
func (a *App) workflowStepOverdue(officeID int64) []wfStepOverdue {
	if a.db == nil || officeID <= 0 {
		return nil
	}
	rows, err := a.db.Query(`SELECT wss.instance_id, wi.request_id, sr.title, svc.name,
		wss.step_key, ws.label, DATE_FORMAT(wss.due_at,'%Y-%m-%d'), DATEDIFF(CURDATE(), wss.due_at)
		FROM workflow_step_states wss
		JOIN workflow_instances wi ON wi.id=wss.instance_id
		JOIN service_requests sr ON sr.id=wi.request_id
		JOIN services svc ON svc.id=sr.service_id
		JOIN workflow_steps ws ON ws.template_id=wi.template_id AND ws.step_key=wss.step_key
		WHERE wi.office_id=? AND wss.status='PENDING'
		AND wss.due_at IS NOT NULL AND wss.due_at < CURDATE()
		ORDER BY wss.due_at LIMIT 20`, officeID)
	if err != nil {
		return nil
	}
	defer rows.Close()
	var out []wfStepOverdue
	for rows.Next() {
		var o wfStepOverdue
		if err := rows.Scan(&o.InstanceID, &o.RequestID, &o.ReqTitle, &o.Service,
			&o.StepKey, &o.Label, &o.DueAt, &o.LateDays); err != nil {
			continue
		}
		out = append(out, o)
	}
	return out
}

// workflowSLAEscalate membuat tugas eskalasi untuk tiap langkah overdue.
// Pola konsisten dengan SLA dokumen (dms.go: hitung OPEN dulu, buat bila
// nol) + retensi (ops_growth.go: 1x per ref, ref bertaut): judul stabil +
// ref_type='request' sehingga klik kedua / hari berikut tidak ganda selama
// tugas masih OPEN. Langkah yang sudah DONE menutup tugasnya (lihat
// workflowStep di bawah, meniru dms.go menutup tugas SLA di status terminal).
func (a *App) workflowSLAEscalate(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	office := a.officeOf(r)
	over := a.workflowStepOverdue(office)
	made := 0
	for _, o := range over {
		title := wfEscalationTitle(o.RequestID, o.StepKey, o.Label)
		var n int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE office_id=? AND status='OPEN' AND ref_type='request' AND ref_id=? AND title=?`,
			office, o.RequestID, title).Scan(&n)
		if n > 0 {
			continue
		}
		if _, err := a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by, ref_type, ref_id)
			VALUES(?,?,CURDATE(),?, 'request', ?)`, office, title, a.uid(r), o.RequestID); err == nil {
			made++
		}
	}
	if made > 0 {
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "workflow.sla.escalate",
			Entity: "office", EntityID: strconv.FormatInt(office, 10),
			Detail: strconv.Itoa(made) + " tugas eskalasi SLA langkah"})
		http.Redirect(w, r, "/tasks?flash=Tugas+eskalasi+SLA+dibuat+"+strconv.Itoa(made), http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/tasks?flash=Tidak+ada+langkah+overdue+baru+(sudah+ada+tugas+OPEN)", http.StatusSeeOther)
}

// workflowActorName menyelesaikan nama aktor untuk audit auto-step:
// nama user bila ada, sonst role. Tak pernah kosong (fallback "sistem").
func (a *App) workflowActorName(r *http.Request) (*int64, string) {
	uid := a.uid(r)
	actor := mw.RoleOf(r)
	if strings.TrimSpace(actor) == "" {
		actor = "sistem"
	}
	if uid != nil && a.db != nil {
		var name sql.NullString
		_ = a.db.QueryRow(`SELECT name FROM users WHERE id=?`, *uid).Scan(&name)
		if name.Valid && strings.TrimSpace(name.String) != "" {
			actor = strings.TrimSpace(name.String)
		}
	}
	return uid, actor
}

// matchStepKey mengembalikan step_key pertama (urut template) yang mengandung
// kw (case-insensitive, substring). Murni tanpa DB/IO; diekstrak mekanis dari
// autoWorkflowStep agar mudah diuji — perilaku identik: contains, bukan
// pencocokan kata (word-boundary). Disengaja: kw "sah" memang harus cocok
// dengan step_key "Sahkan" (awalan), dan word-boundary eksak ("sah"==token)
// justru akan MERUSAK pemetaan itu. Matcher hanya menerima step_key (bukan
// label), jadi label "Pengesahan" tak pernah masuk ke sini — tak ada
// false-positive "Pengesahan"-lebih-dulu di domain nyata (seluruh step_key
// seed diverifikasi di workflow_match_test.go: hanya "Sahkan" yang mengandung
// "sah", dan "Syarat" tidak mengandung "sah" berurutan).
func matchStepKey(keys []string, kw string) string {
	kw = strings.ToLower(strings.TrimSpace(kw))
	if kw == "" {
		return ""
	}
	for _, k := range keys {
		if strings.Contains(strings.ToLower(k), kw) {
			return k
		}
	}
	return ""
}

// autoWorkflowStep menandai step DONE secara best-effort (fail-open):
// cocokkan step_key pertama (urut template) yang mengandung salah satu
// keyword (case-insensitive). Bila tak ada yang cocok / instance tak ada /
// tabel 027 hilang → lewati diam-diam. Bila step yang ditandai adalah
// current_step, current_step dimajukan ke step pending berikut (atau
// COMPLETED), meniru workflowStep manual. Tiap penandaan diaudit
// workflow.step. JANGAN dipanggil sebelum aksi utama sukses; JANGAN
// kembalikan error ke pemanggil (tutup loop engine tanpa blokir alur utama).
//
// Pemetaan kunci (lihat standardWorkflowSeeds):
//   - "draf"   → Draf (deedCreate sukses + request_id ada)
//   - "sah"    → Sahkan (deedApprove sukses)
//   - "syarat" → Syarat (semua checklist VERIFIED/RECEIVED)
//   - "akta"   → Akta (ppatAdvance → AKTA_JADI)
//   - "lapor"  → Lapor (ppatAdvance → LAPOR_BPN)
func (a *App) autoWorkflowStep(officeID, requestID int64, actorID *int64, actor, note string, keywords ...string) {
	if a == nil || a.db == nil || officeID <= 0 || requestID <= 0 || len(keywords) == 0 {
		return
	}
	keys := make([]string, 0, len(keywords))
	for _, k := range keywords {
		if s := strings.ToLower(strings.TrimSpace(k)); s != "" {
			keys = append(keys, s)
		}
	}
	if len(keys) == 0 {
		return
	}
	if strings.TrimSpace(actor) == "" {
		actor = "sistem"
	}
	func() {
		defer func() { _ = recover() }()
		var instID int64
		var cur sql.NullString
		var st string
		if err := a.db.QueryRow(`SELECT id, current_step, status FROM workflow_instances
			WHERE office_id=? AND request_id=?`, officeID, requestID).Scan(&instID, &cur, &st); err != nil {
			return // belum ada instance / tabel hilang: lewati diam-diam
		}
		var tplID int64
		if err := a.db.QueryRow(`SELECT template_id FROM workflow_instances WHERE id=?`, instID).Scan(&tplID); err != nil {
			return
		}
		rows, err := a.db.Query(`SELECT step_key FROM workflow_steps WHERE template_id=? ORDER BY sort_order, step_key`, tplID)
		if err != nil {
			return
		}
		var order []string
		for rows.Next() {
			var k string
			if err := rows.Scan(&k); err == nil {
				order = append(order, k)
			}
		}
		rows.Close()
		if rows.Err() != nil || len(order) == 0 {
			return
		}
		match := ""
		for _, stepKey := range order {
			for _, kw := range keys {
				if matchStepKey([]string{stepKey}, kw) != "" {
					match = stepKey
					break
				}
			}
			if match != "" {
				break
			}
		}
		if match == "" {
			return // template tanpa kunci terkait: lewati diam-diam
		}
		var curStatus string
		_ = a.db.QueryRow(`SELECT COALESCE(status,'PENDING') FROM workflow_step_states
			WHERE instance_id=? AND step_key=?`, instID, match).Scan(&curStatus)
		if strings.EqualFold(strings.TrimSpace(curStatus), "DONE") {
			return // idempoten: sudah DONE, tanpa audit ganda
		}
		if _, err := a.db.Exec(`INSERT INTO workflow_step_states(instance_id, step_key, status, actor, note)
			VALUES(?,?,'DONE',?,?)
			ON DUPLICATE KEY UPDATE status='DONE', actor=VALUES(actor), note=VALUES(note)`,
			instID, match, actor, note); err != nil {
			return
		}
		next := ""
		if cur.Valid && strings.TrimSpace(cur.String) == match && !strings.EqualFold(strings.TrimSpace(st), "COMPLETED") {
			// next = step pending pertama setelah match (DONE dilewati defensif).
			hit := false
			for i, k := range order {
				if k != match {
					continue
				}
				for _, nk := range order[i+1:] {
					var sst string
					_ = a.db.QueryRow(`SELECT COALESCE(status,'PENDING') FROM workflow_step_states
						WHERE instance_id=? AND step_key=?`, instID, nk).Scan(&sst)
					if !strings.EqualFold(strings.TrimSpace(sst), "DONE") {
						next = nk
						break
					}
				}
				hit = true
				break
			}
			if !hit {
				next = ""
			} else if next == "" {
				_, _ = a.db.Exec(`UPDATE workflow_instances SET current_step=NULL, status='COMPLETED' WHERE id=?`, instID)
			} else {
				_, _ = a.db.Exec(`UPDATE workflow_instances SET current_step=?, status='ACTIVE' WHERE id=?`, next, instID)
			}
		}
		a.audit.Record(audit.Entry{ActorID: actorID, Action: "workflow.step", Entity: "service_request",
			EntityID: strconv.FormatInt(requestID, 10), Detail: match + "→DONE (auto: " + strings.TrimSpace(note) + ") oleh " + actor,
			OfficeID: &officeID})
	}()
}

// autoWorkflowSyarat menandai step kunci "syarat" DONE bila SEMUA item
// checklist request itu VERIFIED/RECEIVED dan minimal ada 1 item template.
// Fail-open: tanpa item / tabel hilang / error = lewati diam-diam.
func (a *App) autoWorkflowSyarat(officeID, requestID int64, actorID *int64, actor string) {
	if a == nil || a.db == nil || officeID <= 0 || requestID <= 0 {
		return
	}
	miss, bad, ok := workflowChecklistCounts(a.db, officeID, requestID)
	if !ok || miss > 0 || bad > 0 {
		return
	}
	var total int
	if err := a.db.QueryRow(`SELECT COUNT(*) FROM doc_checklist_items i
		JOIN doc_checklist_templates t ON t.id=i.template_id
		JOIN service_requests sr ON sr.service_id=(SELECT id FROM services WHERE code=t.service_code LIMIT 1)
		WHERE t.office_id=? AND sr.id=? AND sr.office_id=?`, officeID, requestID, officeID).Scan(&total); err != nil || total <= 0 {
		return // tanpa item template: jangan tandai (hindari DONE palsu)
	}
	a.autoWorkflowStep(officeID, requestID, actorID, actor, "checklist lengkap", "syarat")
}

// workflowStep menandai step current DONE (oleh peran yang sesuai) lalu
// memajukan current_step. Tidak mengubah service_requests.status.
func (a *App) workflowStep(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermEditClient) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	office := a.officeOf(r)
	if _, err := a.reqs.Get(office, id); err != nil {
		http.NotFound(w, r)
		return
	}
	var instID, tplID int64
	var cur sql.NullString
	var st string
	if err := a.db.QueryRow(`SELECT id, template_id, current_step, status FROM workflow_instances
		WHERE office_id=? AND request_id=?`, office, id).Scan(&instID, &tplID, &cur, &st); err != nil {
		http.Error(w, "alur kerja belum ada untuk permohonan ini — pasang template dulu via DMS → Checklist → Pasang paket standar", http.StatusNotFound)
		return
	}
	if st == "COMPLETED" || !cur.Valid || strings.TrimSpace(cur.String) == "" {
		http.Redirect(w, r, "/requests/"+strconv.FormatInt(id, 10)+"?flash=Alur+kerja+sudah+selesai", http.StatusSeeOther)
		return
	}
	type row struct {
		key, role string
		skip      int
	}
	var order []row
	rs, err := a.db.Query(`SELECT step_key, required_role, skippable FROM workflow_steps
		WHERE template_id=? ORDER BY sort_order, step_key`, tplID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	for rs.Next() {
		var x row
		_ = rs.Scan(&x.key, &x.role, &x.skip)
		order = append(order, x)
	}
	rs.Close()
	curKey := strings.TrimSpace(cur.String)
	idx := -1
	for i, x := range order {
		if x.key == curKey {
			idx = i
			break
		}
	}
	if idx < 0 {
		http.Error(w, "step berjalan tidak dikenal ("+curKey+") — periksa template", http.StatusConflict)
		return
	}
	// RBAC sederhana: langkah notaris wajib PermApproveDeed; langkah lain
	// cukup PermEditClient (sudah lolos di atas = staf ke atas).
	if req := strings.TrimSpace(order[idx].role); req == rbac.RoleNotary || req == rbac.RoleSubstitute {
		if !rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) {
			a.forbidden(w, r)
			return
		}
	}
	actor := mw.RoleOf(r)
	if uid := a.uid(r); uid != nil {
		var name sql.NullString
		_ = a.db.QueryRow(`SELECT name FROM users WHERE id=?`, *uid).Scan(&name)
		if name.Valid && strings.TrimSpace(name.String) != "" {
			actor = name.String
		}
	}
	// Step berikut yang masih pending (DONE dilewati defensif).
	next := ""
	for _, x := range order[idx+1:] {
		var sst string
		_ = a.db.QueryRow(`SELECT COALESCE(status,'PENDING') FROM workflow_step_states
			WHERE instance_id=? AND step_key=?`, instID, x.key).Scan(&sst)
		if !strings.EqualFold(strings.TrimSpace(sst), "DONE") {
			next = x.key
			break
		}
	}
	tx, err := a.db.Begin()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`INSERT INTO workflow_step_states(instance_id, step_key, status, actor, note)
		VALUES(?,?,'DONE',?,?)
		ON DUPLICATE KEY UPDATE status='DONE', actor=VALUES(actor), note=VALUES(note)`,
		instID, curKey, actor, r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	var flash string
	if next == "" {
		if _, err := tx.Exec(`UPDATE workflow_instances SET current_step=NULL, status='COMPLETED' WHERE id=?`, instID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		flash = "Alur+kerja+selesai"
	} else {
		if _, err := tx.Exec(`UPDATE workflow_instances SET current_step=?, status='ACTIVE' WHERE id=?`, next, instID); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		flash = "Langkah+" + curKey + "+selesai,+lanjut+ke+" + next
	}
	// SLA 033: langkah selesai menutup tugas eskalasinya yang masih OPEN
	// (meniru dms.go menutup tugas SLA di status terminal; best-effort agar
	// tak menggagalkan advance bila kolom 033 belum ada).
	_, _ = tx.Exec(`UPDATE tasks SET status='DONE' WHERE office_id=? AND status='OPEN'
		AND ref_type='request' AND ref_id=? AND title LIKE ?`,
		office, id, "Eskalasi SLA langkah "+curKey+" req #"+strconv.FormatInt(id, 10)+"%")
	if err := tx.Commit(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "workflow.step", Entity: "service_request",
		EntityID: strconv.FormatInt(id, 10), Detail: curKey + "→" + next + " oleh " + actor})
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(id, 10)+"?flash="+flash+"#alurkerja", http.StatusSeeOther)
}
