package app

// Portabilitas template checklist + workflow antar-kantor (027).
//
//   - GET  /dms/checklists/export — unduh JSON office-scoped sebagai
//     attachment (tiru pola header repertoriumExport).
//   - POST /dms/checklists/import — unggah JSON hasil export; validasi
//     struktur + batas 200 item per paket; idempoten: paket dengan
//     (service_code, name) yang sudah ada dilewati.
//
// Format JSON (contoh satu paket):
//
//	{"checklists":[{"service_code":"KUASA","name":"Standar Surat Kuasa",
//	"items":["KTP pemberi & penerima kuasa","Ruang lingkup kuasa"]}],
//	"workflows":[{"service_code":"KUASA","name":"Alur Surat Kuasa",
//	"steps":[{"key":"Intake","label":"Penerimaan","required_role":"staff"}]}]}
//
// Seed existing (seed_catalog.go) tidak diubah.

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"notaryos/internal/audit"
	mw "notaryos/internal/middleware"
	"notaryos/internal/plan"
	"notaryos/internal/rbac"
)

// Batas portabilitas: maks 200 item per paket checklist, maks 100 paket
// per berkas, maks 100 langkah per alur kerja, ukuran berkas maks 2MB.
const (
	checklistPortMaxFile      = 2 << 20
	checklistPortMaxPackages  = 100
	checklistPortMaxItems     = 200
	checklistPortMaxWorkflows = 100
	checklistPortMaxSteps     = 100
)

type checklistStepPort struct {
	Key              string `json:"key"`
	Label            string `json:"label"`
	RequiredRole     string `json:"required_role"`
	RequireChecklist bool   `json:"require_checklist"`
	RequireEvidence  string `json:"require_evidence"`
	Skippable        bool   `json:"skippable"`
}

type checklistPort struct {
	ServiceCode string   `json:"service_code"`
	Name        string   `json:"name"`
	Items       []string `json:"items"`
}

type workflowPort struct {
	ServiceCode string              `json:"service_code"`
	Name        string              `json:"name"`
	Steps       []checklistStepPort `json:"steps"`
}

type checklistPortBundle struct {
	Checklists []checklistPort `json:"checklists"`
	Workflows  []workflowPort  `json:"workflows"`
}

// dmsChecklistsExport mengunduh seluruh template checklist + workflow milik
// kantor aktif sebagai satu JSON (attachment, office-scoped).
func (a *App) dmsChecklistsExport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	if !a.requirePlan(w, r, plan.FeatChecklist) {
		return
	}
	office := a.officeOf(r)
	bundle := checklistPortBundle{}

	trows, err := a.db.Query(`SELECT id, COALESCE(service_code,''), name
		FROM doc_checklist_templates WHERE office_id=? ORDER BY id`, office)
	if err == nil && trows != nil {
		defer trows.Close()
		for trows.Next() {
			var id int64
			var cp checklistPort
			if err := trows.Scan(&id, &cp.ServiceCode, &cp.Name); err != nil {
				continue
			}
			irows, err := a.db.Query(`SELECT label FROM doc_checklist_items
				WHERE template_id=? ORDER BY id`, id)
			if err != nil {
				if irows != nil {
					irows.Close()
				}
				continue
			}
			for irows.Next() {
				var label string
				if err := irows.Scan(&label); err == nil {
					cp.Items = append(cp.Items, label)
				}
			}
			irows.Close()
			if err := irows.Err(); err != nil {
				continue
			}
			bundle.Checklists = append(bundle.Checklists, cp)
		}
		_ = trows.Err()
	}

	// Workflow (tabel 027): absen di DB lama → export checklist saja.
	wrows, err := a.db.Query(`SELECT id, service_code, name
		FROM workflow_templates WHERE office_id=? ORDER BY service_code, version`, office)
	if err == nil && wrows != nil {
		defer wrows.Close()
		for wrows.Next() {
			var id int64
			var wp workflowPort
			if err := wrows.Scan(&id, &wp.ServiceCode, &wp.Name); err != nil {
				continue
			}
			srows, err := a.db.Query(`SELECT step_key, label, required_role,
				require_checklist, COALESCE(require_evidence,''), skippable
				FROM workflow_steps WHERE template_id=? ORDER BY sort_order, step_key`, id)
			if err != nil {
				if srows != nil {
					srows.Close()
				}
				continue
			}
			for srows.Next() {
				var st checklistStepPort
				var rc, sk int
				if err := srows.Scan(&st.Key, &st.Label, &st.RequiredRole,
					&rc, &st.RequireEvidence, &sk); err == nil {
					st.RequireChecklist = rc == 1
					st.Skippable = sk == 1
					wp.Steps = append(wp.Steps, st)
				}
			}
			srows.Close()
			if err := srows.Err(); err != nil {
				continue
			}
			bundle.Workflows = append(bundle.Workflows, wp)
		}
		_ = wrows.Err()
	}

	raw, err := json.MarshalIndent(bundle, "", "  ")
	if err != nil {
		http.Error(w, "gagal menyusun JSON", http.StatusInternalServerError)
		return
	}
	// Pola header download meniru repertoriumExport (attachment).
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="checklists-office-`+
		strconv.FormatInt(office, 10)+`.json"`)
	_, _ = w.Write(raw)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "checklist.export",
		Entity: "office", EntityID: strconv.FormatInt(office, 10),
		Detail: strconv.Itoa(len(bundle.Checklists)) + " paket+" +
			strconv.Itoa(len(bundle.Workflows)) + " alur"})
}

// validateChecklistPortBundle memeriksa struktur + batas ukuran impor.
func validateChecklistPortBundle(b *checklistPortBundle) error {
	if len(b.Checklists) == 0 && len(b.Workflows) == 0 {
		return fmt.Errorf("berkas kosong: butuh minimal 1 checklist atau 1 workflow")
	}
	if len(b.Checklists) > checklistPortMaxPackages {
		return fmt.Errorf("terlalu banyak paket checklist (maks %d)", checklistPortMaxPackages)
	}
	if len(b.Workflows) > checklistPortMaxWorkflows {
		return fmt.Errorf("terlalu banyak alur kerja (maks %d)", checklistPortMaxWorkflows)
	}
	for i := range b.Checklists {
		c := &b.Checklists[i]
		c.ServiceCode = strings.TrimSpace(c.ServiceCode)
		c.Name = strings.TrimSpace(c.Name)
		if c.ServiceCode == "" || c.Name == "" {
			return fmt.Errorf("checklists[%d]: service_code dan name wajib diisi", i)
		}
		if len(c.ServiceCode) > 50 || len(c.Name) > 255 {
			return fmt.Errorf("checklists[%d]: service_code (maks 50) / name (maks 255) terlalu panjang", i)
		}
		if len(c.Items) == 0 {
			return fmt.Errorf("checklists[%d] (%s): minimal 1 item", i, c.Name)
		}
		if len(c.Items) > checklistPortMaxItems {
			return fmt.Errorf("checklists[%d] (%s): maks %d item", i, c.Name, checklistPortMaxItems)
		}
		for j := range c.Items {
			c.Items[j] = strings.TrimSpace(c.Items[j])
			if c.Items[j] == "" {
				return fmt.Errorf("checklists[%d] (%s): item ke-%d kosong", i, c.Name, j+1)
			}
			if len(c.Items[j]) > 255 {
				return fmt.Errorf("checklists[%d] (%s): item ke-%d terlalu panjang (maks 255)", i, c.Name, j+1)
			}
		}
	}
	for i := range b.Workflows {
		wf := &b.Workflows[i]
		wf.ServiceCode = strings.TrimSpace(wf.ServiceCode)
		wf.Name = strings.TrimSpace(wf.Name)
		if wf.ServiceCode == "" || wf.Name == "" {
			return fmt.Errorf("workflows[%d]: service_code dan name wajib diisi", i)
		}
		if len(wf.ServiceCode) > 50 || len(wf.Name) > 255 {
			return fmt.Errorf("workflows[%d]: service_code (maks 50) / name (maks 255) terlalu panjang", i)
		}
		if len(wf.Steps) == 0 {
			return fmt.Errorf("workflows[%d] (%s): minimal 1 langkah", i, wf.Name)
		}
		if len(wf.Steps) > checklistPortMaxSteps {
			return fmt.Errorf("workflows[%d] (%s): maks %d langkah", i, wf.Name, checklistPortMaxSteps)
		}
		for j := range wf.Steps {
			st := &wf.Steps[j]
			st.Key = strings.TrimSpace(st.Key)
			st.Label = strings.TrimSpace(st.Label)
			st.RequiredRole = strings.TrimSpace(st.RequiredRole)
			st.RequireEvidence = strings.TrimSpace(st.RequireEvidence)
			if st.Key == "" || st.Label == "" {
				return fmt.Errorf("workflows[%d] (%s): langkah ke-%d butuh key dan label", i, wf.Name, j+1)
			}
			if len(st.Key) > 100 || len(st.Label) > 255 || len(st.RequiredRole) > 50 || len(st.RequireEvidence) > 50 {
				return fmt.Errorf("workflows[%d] (%s): langkah ke-%d terlalu panjang", i, wf.Name, j+1)
			}
			if st.RequiredRole == "" {
				st.RequiredRole = "staff"
			}
		}
	}
	return nil
}

// dmsChecklistsImport mengunggah JSON hasil export antar-kantor.
// Idempoten: paket (service_code, name) yang sudah ada dilewati.
func (a *App) dmsChecklistsImport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	if !a.requirePlan(w, r, plan.FeatChecklist) {
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, checklistPortMaxFile+(1<<20))
	var bundle checklistPortBundle
	if f, _, err := r.FormFile("file"); err == nil {
		defer f.Close()
		if err := json.NewDecoder(f).Decode(&bundle); err != nil {
			http.Error(w, "berkas bukan JSON valid: "+err.Error(), http.StatusBadRequest)
			return
		}
	} else if err := json.NewDecoder(r.Body).Decode(&bundle); err != nil {
		http.Error(w, "unggah berkas JSON via kolom file (atau body JSON): "+err.Error(), http.StatusBadRequest)
		return
	}
	if err := validateChecklistPortBundle(&bundle); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	office := a.officeOf(r)
	madeCL, skipCL, madeWF, skipWF := 0, 0, 0, 0
	for _, c := range bundle.Checklists {
		var n int
		if err := a.db.QueryRow(`SELECT COUNT(*) FROM doc_checklist_templates
			WHERE office_id=? AND service_code=? AND name=?`,
			office, c.ServiceCode, c.Name).Scan(&n); err == nil && n > 0 {
			skipCL++
			continue
		}
		if _, err := a.dms.CreateChecklistTemplate(office, c.ServiceCode, c.Name, c.Items); err != nil {
			http.Error(w, "gagal menyimpan paket "+c.Name+": "+err.Error(), http.StatusBadRequest)
			return
		}
		madeCL++
	}
	for _, wf := range bundle.Workflows {
		seed := workflowSeed{code: wf.ServiceCode, name: wf.Name}
		for _, st := range wf.Steps {
			seed.steps = append(seed.steps, workflowStepSeed{
				key: st.Key, label: st.Label, requiredRole: st.RequiredRole,
				requireChecklist: st.RequireChecklist,
				requireEvidence:  st.RequireEvidence,
				skippable:        st.Skippable,
			})
		}
		var tplID int64
		err := a.db.QueryRow(`SELECT id FROM workflow_templates
			WHERE office_id=? AND service_code=? AND version=1`,
			office, wf.ServiceCode).Scan(&tplID)
		if err == nil {
			// Idempoten per (service_code, name): template service yang
			// sudah ada dilewati, step yang kurang di-backfill best-effort.
			var nm string
			if err := a.db.QueryRow(`SELECT name FROM workflow_templates WHERE id=?`,
				tplID).Scan(&nm); err == nil && strings.TrimSpace(nm) == wf.Name {
				a.ensureWorkflowSteps(tplID, seed)
				skipWF++
				continue
			}
			skipWF++
			continue
		}
		if err != sql.ErrNoRows {
			if isTableMissing(err) {
				break // DB lama tanpa tabel 027: checklist tetap tersimpan
			}
			http.Error(w, "gagal memeriksa alur "+wf.Name+": "+err.Error(), http.StatusInternalServerError)
			return
		}
		res, err := a.db.Exec(`INSERT INTO workflow_templates(office_id, service_code, name, version, active)
			VALUES(?,?,?,1,1)`, office, wf.ServiceCode, wf.Name)
		if err != nil {
			if isTableMissing(err) {
				break
			}
			http.Error(w, "gagal menyimpan alur "+wf.Name+": "+err.Error(), http.StatusBadRequest)
			return
		}
		tplID, _ = res.LastInsertId()
		if tplID == 0 {
			http.Error(w, "gagal menyimpan alur "+wf.Name, http.StatusInternalServerError)
			return
		}
		a.ensureWorkflowSteps(tplID, seed)
		madeWF++
	}
	detail := strconv.Itoa(madeCL) + " paket+" + strconv.Itoa(madeWF) + " alur baru, " +
		strconv.Itoa(skipCL+skipWF) + " dilewati (sudah ada)"
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "checklist.import",
		Entity: "office", EntityID: strconv.FormatInt(office, 10), Detail: detail})
	http.Redirect(w, r, "/dms/checklists?flash="+url.QueryEscape("Impor selesai: "+detail), http.StatusSeeOther)
}
