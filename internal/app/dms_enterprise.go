package app

// DMS enterprise: custom fields, advanced search, bulk ops, version compare,
// approval matrix, watermark, quotas, office export, confidentiality, analytics.

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/diff"
	"notaryos/internal/dms"
	mw "notaryos/internal/middleware"
	"notaryos/internal/officex"
	"notaryos/internal/rbac"
)

// canSeeDoc enforces confidentiality server-side (never rely on hidden links).
func (a *App) canSeeDoc(r *http.Request, conf string) bool {
	if conf == "" {
		conf = "INTERNAL"
	}
	if conf == "PUBLIC" || conf == "INTERNAL" {
		role := mw.RoleOf(r)
		return role != "" && role != "client"
	}
	return dms.CanAccess(mw.RoleOf(r), conf)
}

func dmsCanSee(role, conf string) bool {
	if conf == "" {
		conf = "INTERNAL"
	}
	if conf == "PUBLIC" || conf == "INTERNAL" {
		return role != "" && role != "client"
	}
	return dms.CanAccess(role, conf)
}

// ---- advanced search ----

func (a *App) dmsSearchAdvanced(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	q := r.URL.Query()
	type hit struct {
		ID, Size                    int64
		Name, MIME, OCR, Ver, WF    string
		Conf                        string
	}
	var out []hit
	if strings.TrimSpace(q.Get("q")) != "" || q.Get("wf") != "" || q.Get("mime") != "" {
		query := `SELECT f.id, COALESCE(f.display_name,f.original_name), f.mime, f.size_bytes,
			f.ocr_status, f.verification_status, f.workflow_status, f.confidentiality FROM files f WHERE f.office_id=?`
		args := []any{a.officeOf(r)}
		if t := strings.TrimSpace(q.Get("q")); t != "" {
			like := "%" + t + "%"
			query += ` AND (f.original_name LIKE ? OR COALESCE(f.display_name,'') LIKE ? OR f.doc_number LIKE ? OR f.id IN
				(SELECT file_id FROM doc_search_index WHERE MATCH(content) AGAINST(? IN NATURAL LANGUAGE MODE)))`
			args = append(args, like, like, like, t)
		}
		for col, key := range map[string]string{"f.workflow_status": "wf", "f.verification_status": "vst", "f.confidentiality": "conf", "f.mime": "mime", "f.ocr_status": "ocr"} {
			if v := q.Get(key); v != "" {
				query += ` AND ` + col + `=?`
				args = append(args, v)
			}
		}
		if v := q.Get("from"); v != "" {
			query += ` AND DATE(f.created_at)>=?`
			args = append(args, v)
		}
		if v := q.Get("to"); v != "" {
			query += ` AND DATE(f.created_at)<=?`
			args = append(args, v)
		}
		sort := q.Get("sort")
		switch sort {
		case "size_desc":
			query += ` ORDER BY f.size_bytes DESC`
		case "size_asc":
			query += ` ORDER BY f.size_bytes ASC`
		case "name":
			query += ` ORDER BY COALESCE(f.display_name,f.original_name)`
		default:
			query += ` ORDER BY f.id DESC`
		}
		query += ` LIMIT 100`
		rows, _ := a.db.Query(query, args...)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var h hit
				_ = rows.Scan(&h.ID, &h.Name, &h.MIME, &h.Size, &h.OCR, &h.Ver, &h.WF, &h.Conf)
				if dmsCanSee(mw.RoleOf(r), h.Conf) {
					out = append(out, h)
				}
			}
		}
	}
	a.render(w, r, "dms_search_adv", "Pencarian Lanjutan", map[string]any{
		"Hits": out, "Q": q.Get("q"), "WF": q.Get("wf"), "VST": q.Get("vst"),
		"Conf": q.Get("conf"), "MIME": q.Get("mime"), "OCR": q.Get("ocr"),
		"From": q.Get("from"), "To": q.Get("to"), "Sort": q.Get("sort"),
	})
}

// ---- custom fields ----

func (a *App) dmsFields(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	fields, _ := a.dms.CustomFields(a.officeOf(r))
	a.render(w, r, "dms_fields", "Field Kustom", map[string]any{"Fields": fields})
}

func (a *App) dmsFieldSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	pos, _ := strconv.Atoi(r.FormValue("position"))
	_, err := a.dms.SaveCustomField(a.officeOf(r), dmsFieldOf(r, pos))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.field.define", Entity: "office", EntityID: strconv.FormatInt(a.officeOf(r), 10), Detail: r.FormValue("name")})
	http.Redirect(w, r, "/dms/fields?flash=Field+disimpan", http.StatusSeeOther)
}

func dmsFieldOf(r *http.Request, pos int) dms.CustomField {
	req := r.FormValue("required") == "1"
	return dms.CustomField{Name: strings.TrimSpace(r.FormValue("name")), Type: r.FormValue("ftype"),
		Options: strings.TrimSpace(r.FormValue("options")), Required: req, Position: pos}
}

// ---- bulk ops ----

func (a *App) dmsBulk(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	action := r.FormValue("action")
	ids := r.Form["ids"]
	if len(ids) == 0 || (action != "verify" && action != "approve") {
		http.Error(w, "pilih berkas dan aksi yang valid", http.StatusBadRequest)
		return
	}
	n := 0
	for _, s := range ids {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil || id <= 0 {
			continue
		}
		var conf string
		_ = a.db.QueryRow(`SELECT confidentiality FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&conf)
		if !dmsCanSee(mw.RoleOf(r), conf) {
			continue
		}
		switch action {
		case "verify":
			if _, err := a.db.Exec(`UPDATE files SET verification_status='VERIFIED' WHERE id=? AND office_id=?`, id, a.officeOf(r)); err == nil {
				n++
			}
		case "approve":
			if err := a.dms.Move(a.officeOf(r), id, "APPROVED", a.uid(r), "bulk"); err == nil {
				n++
			}
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.bulk." + action, Entity: "file", EntityID: "bulk", Detail: fmt.Sprintf("%d berkas", n)})
	http.Redirect(w, r, "/dms/library?flash="+strconv.Itoa(n)+"+berkas+diproses", http.StatusSeeOther)
}

// dmsLibraryExport streams a CSV of the library (same filters as list, confidentiality-scoped).
func (a *App) dmsLibraryExport(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	rows, _ := a.db.Query(`SELECT id, COALESCE(display_name,original_name), mime, size_bytes, verification_status, workflow_status, confidentiality, created_at
		FROM files WHERE office_id=? ORDER BY id DESC LIMIT 2000`, a.officeOf(r))
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"library.csv\"")
	_, _ = io.WriteString(w, "id,nama,mime,ukuran,verifikasi,workflow,akses,diunggah\n")
	if rows == nil {
		return
	}
	defer rows.Close()
	for rows.Next() {
		var id, size int64
		var name, mime, ver, wf, conf, at string
		_ = rows.Scan(&id, &name, &mime, &size, &ver, &wf, &conf, &at)
		if !dmsCanSee(mw.RoleOf(r), conf) {
			continue
		}
		name = strings.ReplaceAll(strings.ReplaceAll(name, `"`, "'"), "\n", " ")
		fmt.Fprintf(w, "%d,\"%s\",%s,%d,%s,%s,%s,%s\n", id, name, mime, size, ver, wf, conf, at)
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.library.export", Entity: "file", EntityID: "bulk"})
}

// ---- compare versions ----

type cmpVer struct {
	No              int
	Note, Hash      string
	Size            int64
	CreatedAt       string
	Ref             string
}

func (a *App) dmsCompare(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var fname, fmime string
	_ = a.db.QueryRow(`SELECT COALESCE(display_name, original_name), mime FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&fname, &fmime)
	aQ, bQ := r.URL.Query().Get("a"), r.URL.Query().Get("b")
	vers, _ := a.dms.Versions(a.officeOf(r), id)
	var va, vb *cmpVer
	for i := range vers {
		v := vers[i]
		c := &cmpVer{No: v.No, Note: v.Note, Hash: v.Hash, Size: v.Size, CreatedAt: v.At, Ref: v.Ref}
		if strconv.Itoa(v.No) == aQ {
			va = c
		}
		if strconv.Itoa(v.No) == bQ {
			vb = c
		}
	}
	type row struct{ Kind, Line string }
	var ops []row
	var summary, nota, notb string
	if va != nil && vb != nil {
		ta := a.versionText(va.Ref, fmime, fname)
		tb := a.versionText(vb.Ref, fmime, fname)
		nota, notb = va.Note, vb.Note
		if ta.ok && tb.ok {
			dops := diff.Compare(ta.text, tb.text)
			for _, o := range dops {
				ops = append(ops, row{string(o.Kind), o.Line})
			}
			add, del := diff.Stats(dops)
			summary = fmt.Sprintf("+%d -%d baris", add, del)
		} else {
			summary = "Banding-banding biner: " + binCompare(va, vb)
		}
	}
	a.render(w, r, "dms_compare", "Bandingkan Versi", map[string]any{
		"ID": id, "Vers": vers, "A": aQ, "B": bQ, "Ops": ops, "Summary": summary,
		"NoteA": nota, "NoteB": notb,
	})
}

type textOut struct {
	text string
	ok   bool
}

func (a *App) versionText(ref, mime, name string) textOut {
	rc, err := a.files.Open(ref)
	if err != nil {
		return textOut{}
	}
	defer rc.Close()
	buf := make([]byte, 0, 1<<20)
	tmp := make([]byte, 32<<10)
	for {
		n, rerr := rc.Read(tmp)
		if n > 0 {
			buf = append(buf, tmp[:n]...)
			if len(buf) > 4<<20 {
				break
			}
		}
		if rerr != nil {
			break
		}
	}
	if len(buf) == 0 {
		return textOut{}
	}
	lower := strings.ToLower(name)
	if strings.HasSuffix(lower, ".txt") || strings.HasSuffix(lower, ".md") || mime == "text/plain" {
		return textOut{text: string(buf), ok: true}
	}
	t, ok := officex.Extract(name, mime, bytes.NewReader(buf), int64(len(buf)))
	if !ok {
		return textOut{}
	}
	return textOut{text: t, ok: true}
}

func binCompare(a, b *cmpVer) string {
	if a.Hash == b.Hash {
		return "identik (SHA-256 sama)"
	}
	return fmt.Sprintf("berbeda: %d vs %d bytes", a.Size, b.Size)
}

// dmsFieldsSave stores per-document custom field values (validated).
func (a *App) dmsFieldsSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	vals := map[int64]string{}
	for k, v := range r.Form {
		var fid int64
		if _, err := fmt.Sscanf(k, "field_%d", &fid); err != nil || fid <= 0 || len(v) == 0 {
			continue
		}
		vals[fid] = v[0]
	}
	if err := a.dms.SetFieldValues(a.officeOf(r), id, vals); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.fields.save", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Field+disimpan#meta", http.StatusSeeOther)
}

// ---- approval matrix ----

func (a *App) dmsChain(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	chain, _ := a.dms.ApprovalChain(a.officeOf(r))
	a.render(w, r, "dms_chain", "Rantai Persetujuan", map[string]any{"Chain": chain})
}

func (a *App) dmsChainSave(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	var roles []string
	for i := 1; i <= 5; i++ {
		if v := strings.TrimSpace(r.FormValue(fmt.Sprintf("role%d", i))); v != "" {
			roles = append(roles, v)
		}
	}
	if err := a.dms.SaveChain(a.officeOf(r), roles); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.chain.save", Entity: "office", EntityID: strconv.FormatInt(a.officeOf(r), 10), Detail: strings.Join(roles, ">")})
	http.Redirect(w, r, "/dms/chain?flash=Rantai+disimpan", http.StatusSeeOther)
}

func (a *App) dmsApproveStep(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	step, _ := strconv.Atoi(r.FormValue("step"))
	// Caller must satisfy the step's required role (server-side, not UI).
	var need string
	_ = a.db.QueryRow(`SELECT role_required FROM doc_approvals WHERE file_id=? AND step_order=? AND office_id=?`,
		id, step, a.officeOf(r)).Scan(&need)
	if need == "" || !rbac.Has(mw.RoleOf(r), need) {
		a.forbidden(w, r)
		return
	}
	uid := int64(0)
	if u := a.uid(r); u != nil {
		uid = *u
	}
	approve := r.FormValue("decision") != "reject"
	if err := a.dms.DecideApproval(a.officeOf(r), id, step, uid, approve, r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.approve.step", Entity: "file",
		EntityID: strconv.FormatInt(id, 10), Detail: fmt.Sprintf("langkah %d %s", step, map[bool]string{true: "setuju", false: "tolak"}[approve])})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Keputusan+dicatat", http.StatusSeeOther)
}

// chainSatisfied reports whether all approval steps are APPROVED (or no chain).
func (a *App) chainSatisfied(officeID, fileID int64) bool {
	pending, total := a.dms.ApprovalState(fileID)
	_ = officeID
	return total == 0 || pending == 0
}

// ---- watermarked preview (overlay, non-destructive) + audit ----

func (a *App) dmsPreviewWM(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var conf, name string
	if err := a.db.QueryRow(`SELECT confidentiality, COALESCE(display_name, original_name) FROM files WHERE id=? AND office_id=?`,
		id, a.officeOf(r)).Scan(&conf, &name); err != nil {
		http.NotFound(w, r)
		return
	}
	if !dmsCanSee(mw.RoleOf(r), conf) {
		a.forbidden(w, r)
		return
	}
	uname := ""
	if u := a.uid(r); u != nil {
		_ = a.db.QueryRow(`SELECT name FROM users WHERE id=?`, *u).Scan(&uname)
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.preview.wm", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	a.render(w, r, "dms_preview_wm", "Pratinjau Ber-Watermark", map[string]any{
		"ID": id, "Name": name, "User": uname,
	})
}

// ---- retention auto-notify (runs inside daily reminders) ----

func (a *App) notifyExpiring(officeID int64) int {
	n := 0
	rows, _ := a.db.Query(`SELECT id, COALESCE(display_name, original_name), expiry_date FROM files
		WHERE office_id=? AND expiry_date IS NOT NULL AND expiry_date <= DATE_ADD(CURDATE(), INTERVAL 30 DAY)
		AND workflow_status NOT IN ('ARCHIVED','FINALIZED') LIMIT 50`, officeID)
	if rows == nil {
		return 0
	}
	defer rows.Close()
	type f struct {
		ID, Name, Exp string
	}
	var list []f
	for rows.Next() {
		var x f
		var id int64
		_ = rows.Scan(&id, &x.Name, &x.Exp)
		x.ID = strconv.FormatInt(id, 10)
		list = append(list, x)
	}
	if len(list) == 0 {
		return 0
	}
	var staff []int64
	srows, _ := a.db.Query(`SELECT u.id FROM users u JOIN roles r ON r.id=u.role_id
		WHERE u.office_id=? AND u.is_active=1 AND r.name IN ('super_admin','office_admin','notary')`, officeID)
	if srows != nil {
		defer srows.Close()
		for srows.Next() {
			var uid int64
			if err := srows.Scan(&uid); err == nil {
				staff = append(staff, uid)
			}
		}
	}
	for _, x := range list {
		for _, uid := range staff {
			var dup int
			_ = a.db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND title='Retensi mendekati' AND body LIKE ? AND created_at > DATE_SUB(UTC_TIMESTAMP(), INTERVAL 7 DAY)`,
				uid, "%"+x.ID+"%").Scan(&dup)
			if dup == 0 {
				_, _ = a.db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?, 'Retensi mendekati', ?)`,
					uid, "Berkas #"+x.ID+" "+x.Name+" kedaluwarsa "+x.Exp)
				n++
			}
		}
		a.audit.Record(audit.Entry{ActorID: nil, Action: "retention.notice", Entity: "file", EntityID: x.ID})
	}
	return n
}

// ---- storage quota ----

// overQuota reports whether adding `add` bytes would exceed the office quota.
func (a *App) overQuota(officeID int64, add int64) bool {
	var quota int64
	_ = a.db.QueryRow(`SELECT storage_quota_bytes FROM offices WHERE id=?`, officeID).Scan(&quota)
	if quota <= 0 {
		return false
	}
	return a.dms.StorageUsage(officeID)+add > quota
}

// ---- office data export (ZIP: manifest + CSVs; binaries included, capped) ----

func (a *App) officeExport(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageOffice) {
		a.forbidden(w, r)
		return
	}
	office := a.officeOf(r)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="kantor-export.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()
	writeCSV := func(name string, query string, args ...any) {
		fw, err := zw.Create(name)
		if err != nil {
			return
		}
		rows, err := a.db.Query(query, args...)
		if err != nil {
			return
		}
		defer rows.Close()
		cols, _ := rows.Columns()
		fmt.Fprintf(fw, "%s\n", strings.Join(cols, ","))
		vals := make([]any, len(cols))
		ptrs := make([]any, len(cols))
		for i := range vals {
			ptrs[i] = &vals[i]
		}
		for rows.Next() {
			_ = rows.Scan(ptrs...)
			var cells []string
			for _, v := range vals {
				s := fmt.Sprintf("%v", v)
				if strings.ContainsAny(s, ",\"\n") {
					s = `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
				}
				cells = append(cells, s)
			}
			fmt.Fprintln(fw, strings.Join(cells, ","))
		}
	}
	writeCSV("clients.csv", `SELECT id, kind, full_name, email, phone, address, npwp, reg_number, city, province, status, created_at FROM clients WHERE office_id=?`, office)
	writeCSV("requests.csv", `SELECT id, client_id, service_id, status, title, created_at FROM service_requests WHERE office_id=?`, office)
	writeCSV("deeds.csv", `SELECT id, number, title, kind, status, version, created_at FROM deed_drafts WHERE office_id=?`, office)
	writeCSV("invoices.csv", `SELECT id, number, client_id, total, paid, status, created_at FROM invoices WHERE office_id=?`, office)
	writeCSV("files.csv", `SELECT id, owner_type, owner_id, original_name, mime, size_bytes, sha256, workflow_status, verification_status, created_at FROM files WHERE office_id=?`, office)
	type fr struct{ ref, name string }
	var refs []fr
	if rows, err := a.db.Query(`SELECT ref, original_name FROM files WHERE office_id=? ORDER BY id`, office); err == nil {
		defer rows.Close()
		for rows.Next() {
			var x fr
			_ = rows.Scan(&x.ref, &x.name)
			refs = append(refs, x)
		}
	}
	var total int64
	for _, x := range refs {
		rc, err := a.files.Open(x.ref)
		if err != nil {
			continue
		}
		fw, err := zw.Create("files/" + x.ref)
		if err != nil {
			rc.Close()
			continue
		}
		n, _ := copyCapped(fw, rc, 100<<20-total)
		rc.Close()
		total += n
		if total >= 500<<20 {
			break
		}
	}
	mf, _ := zw.Create("MANIFEST.txt")
	fmt.Fprintf(mf, "Ekspor kantor %d — dibuat otomatis.\nBerisi CSV + berkas (maks 500MB).\n", office)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "office.export", Entity: "office", EntityID: strconv.FormatInt(office, 10)})
}

func copyCapped(w io.Writer, r io.Reader, cap int64) (int64, error) {
	if cap <= 0 {
		return 0, nil
	}
	buf := make([]byte, 32<<10)
	var n int64
	for n < cap {
		m := int64(len(buf))
		if n+m > cap {
			m = cap - n
		}
		k, err := r.Read(buf[:m])
		if k > 0 {
			if _, werr := w.Write(buf[:k]); werr != nil {
				return n, werr
			}
			n += int64(k)
		}
		if err != nil {
			break
		}
	}
	return n, nil
}
