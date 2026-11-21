package app

// DMS HTTP layer: library, detail+preview, versions, workflow, OCR queue,
// shares, archive, retention, checklist, admin. All office-scoped, audited.

import (
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

func clientIPOf(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

func (a *App) requireStaff(w http.ResponseWriter, r *http.Request) bool {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewClient) {
		a.forbidden(w, r)
		return false
	}
	return true
}

// ---- dashboard ----

func (a *App) dmsDashboard(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	office := a.officeOf(r)
	st, _ := a.dms.Stats(office)
	recent, _ := a.db.Query(`SELECT f.id, COALESCE(f.display_name,f.original_name), f.workflow_status, f.created_at
		FROM files f WHERE f.office_id=? ORDER BY f.id DESC LIMIT 10`, office)
	type row struct {
		ID           int64
		Name, WF, At string
	}
	var rows []row
	if recent != nil {
		defer recent.Close()
		for recent.Next() {
			var x row
			_ = recent.Scan(&x.ID, &x.Name, &x.WF, &x.At)
			rows = append(rows, x)
		}
	}
	dups, _ := a.dms.DuplicateCandidates(office)
	exp, _ := a.dms.Expiring(office)
	jobs, _ := a.dms.OCRQueue(office)
	pendWF, _ := a.db.Query(`SELECT id, COALESCE(display_name,original_name), workflow_status FROM files
		WHERE office_id=? AND workflow_status IN ('ADMIN_REVIEW','VERIFICATION','APPROVAL') ORDER BY id DESC LIMIT 10`, office)
	var pwf []row
	if pendWF != nil {
		defer pendWF.Close()
		for pendWF.Next() {
			var x row
			_ = pendWF.Scan(&x.ID, &x.Name, &x.WF, &x.At)
			pwf = append(pwf, x)
		}
	}
	a.render(w, r, "dms_dashboard", "DMS Overview", map[string]any{
		"Stats": st, "Recent": rows, "Dups": dups, "Expiring": exp, "Jobs": jobs, "PendingWF": pwf,
	})
}

// ---- library ----

func (a *App) dmsLibrary(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	q := r.URL.Query()
	like := "%" + q.Get("q") + "%"
	cat, _ := strconv.ParseInt(q.Get("cat"), 10, 64)
	wf, vst, conf := q.Get("wf"), q.Get("vst"), q.Get("conf")
	fav := q.Get("fav") == "1"
	query := `SELECT f.id, COALESCE(f.display_name,f.original_name), f.mime, f.size_bytes,
		f.ocr_status, f.verification_status, f.workflow_status, f.created_at FROM files f WHERE f.office_id=?`
	args := []any{a.officeOf(r)}
	if qq := strings.TrimSpace(q.Get("q")); qq != "" {
		query += ` AND (f.original_name LIKE ? OR COALESCE(f.display_name,'') LIKE ? OR f.doc_number LIKE ?)`
		args = append(args, like, like, like)
	}
	if cat != 0 {
		query += ` AND f.category_id=?`
		args = append(args, cat)
	}
	if wf != "" {
		query += ` AND f.workflow_status=?`
		args = append(args, wf)
	}
	if vst != "" {
		query += ` AND f.verification_status=?`
		args = append(args, vst)
	}
	if conf != "" {
		query += ` AND f.confidentiality=?`
		args = append(args, conf)
	}
	if fav {
		if id := a.uid(r); id != nil {
			query += ` AND EXISTS(SELECT 1 FROM doc_favorites WHERE user_id=? AND file_id=f.id)`
			args = append(args, *id)
		}
	}
	query += ` ORDER BY f.id DESC LIMIT 100`
	rows, _ := a.db.Query(query, args...)
	type doc struct {
		ID, Size                     int64
		Name, MIME, OCR, Ver, WF, At string
	}
	var out []doc
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var d doc
			_ = rows.Scan(&d.ID, &d.Name, &d.MIME, &d.Size, &d.OCR, &d.Ver, &d.WF, &d.At)
			out = append(out, d)
		}
	}
	cats, _ := a.dms.Categories(a.officeOf(r))
	a.render(w, r, "dms_library", "Document Library", map[string]any{
		"Docs": out, "Cats": cats, "Q": q.Get("q"), "Cat": cat,
		"WF": wf, "VST": vst, "Conf": conf, "Fav": fav,
	})
}

// ---- detail ----

func (a *App) dmsDetail(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var m struct {
		ID, Size, CatID                int64
		Name, Disp, MIME, OCR, Ver, WF string
		Num, DocDate, Exp              string
		Conf                           string
		At                             string
	}
	var disp, num, ddate, exp sql.NullString
	var catID sql.NullInt64
	err := a.db.QueryRow(`SELECT id, original_name, display_name, mime, size_bytes, ocr_status,
		verification_status, workflow_status, doc_number, doc_date, expiry_date, confidentiality, category_id, created_at
		FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).
		Scan(&m.ID, &m.Name, &disp, &m.MIME, &m.Size, &m.OCR, &m.Ver, &m.WF, &num, &ddate, &exp, &m.Conf, &catID, &m.At)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if disp.Valid {
		m.Disp = disp.String
	}
	if num.Valid {
		m.Num = num.String
	}
	if ddate.Valid {
		m.DocDate = ddate.String
	}
	if exp.Valid {
		m.Exp = exp.String
	}
	if catID.Valid {
		m.CatID = catID.Int64
	}
	vers, _ := a.dms.Versions(a.officeOf(r), id)
	rels, _ := a.dms.RelationsOf(a.officeOf(r), id)
	tl, _ := a.dms.Timeline(a.officeOf(r), id)
	holds, _ := a.dms.ActiveHolds(a.officeOf(r), id)
	hist, _ := a.dms.MoveHistory(a.officeOf(r), id)
	alog, _ := a.dms.AccessLog(a.officeOf(r), id)
	ocrs, _ := a.uploads.OCRResults(id)
	cats, _ := a.dms.Categories(a.officeOf(r))
	locs, _ := a.dms.Locations(a.officeOf(r))
	a.render(w, r, "dms_detail", "Dokumen #"+strconv.FormatInt(id, 10), map[string]any{
		"M": m, "Vers": vers, "Rels": rels, "Timeline": tl, "Holds": holds,
		"Hist": hist, "Access": alog, "OCRs": ocrs, "Cats": cats, "Locs": locs,
	})
}

func (a *App) dmsSaveMeta(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var oid int64
	if err := a.db.QueryRow(`SELECT office_id FROM files WHERE id=?`, id).Scan(&oid); err != nil || oid != a.officeOf(r) {
		http.NotFound(w, r)
		return
	}
	var cat any
	if v := r.FormValue("category_id"); v != "" {
		cat = v
	}
	_, err := a.db.Exec(`UPDATE files SET display_name=?, doc_number=?, category_id=?, doc_date=?, expiry_date=?, confidentiality=? WHERE id=?`,
		nullStrOpt(r.FormValue("display_name")), nullStrOpt(r.FormValue("doc_number")), cat,
		nullStrOpt(r.FormValue("doc_date")), nullStrOpt(r.FormValue("expiry_date")), r.FormValue("confidentiality"), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.dms.LogAccess(id, a.uid(r), "VIEW", clientIPOf(r), r.UserAgent())
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.meta", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Metadata+disimpan", http.StatusSeeOther)
}

func nullStrOpt(s string) any {
	if strings.TrimSpace(s) == "" {
		return nil
	}
	return s
}

func (a *App) dmsVerify(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermVerifyIdentity) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := r.FormValue("to")
	if to != "VERIFIED" && to != "REJECTED" {
		http.Error(w, "status verifikasi tidak dikenal", http.StatusBadRequest)
		return
	}
	var oid int64
	if err := a.db.QueryRow(`SELECT office_id FROM files WHERE id=?`, id).Scan(&oid); err != nil || oid != a.officeOf(r) {
		http.NotFound(w, r)
		return
	}
	if _, err := a.db.Exec(`UPDATE files SET verification_status=? WHERE id=?`, to, id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.verify." + strings.ToLower(to), Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("note")})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Verifikasi+dicatat", http.StatusSeeOther)
}

// ---- preview (inline, authorized, logged) ----

func (a *App) dmsPreview(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	m, rc, err := a.uploads.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	if m.MIME != "application/pdf" && !strings.HasPrefix(m.MIME, "image/") {
		http.Error(w, "pratinjau hanya untuk PDF/gambar; unduh untuk format lain", http.StatusUnsupportedMediaType)
		return
	}
	a.dms.LogAccess(id, a.uid(r), "PREVIEW", clientIPOf(r), r.UserAgent())
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.preview", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	w.Header().Set("Content-Type", m.MIME)
	w.Header().Set("Content-Disposition", "inline")
	_, _ = io.Copy(w, rc)
}

// ---- versions ----

func (a *App) dmsNewVersion(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseMultipartForm(100 << 20)
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "pilih file terlebih dahulu", http.StatusBadRequest)
		return
	}
	defer f.Close()
	nid, err := a.uploads.SaveUpload(a.officeOf(r), "doc_version", id, hdr.Filename, hdr.Header.Get("Content-Type"), f, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Snapshot new binary as next version of the original, then drop the temp row.
	var ref, sha string
	var size int64
	_ = a.db.QueryRow(`SELECT ref, sha256, size_bytes FROM files WHERE id=?`, nid).Scan(&ref, &sha, &size)
	if n, err := a.dms.AddVersion(a.officeOf(r), id, ref, sha, size, r.FormValue("note"), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	} else {
		_ = n
	}
	_, _ = a.db.Exec(`DELETE FROM files WHERE id=?`, nid)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.version", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("note")})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Versi+baru+tersimpan", http.StatusSeeOther)
}

func (a *App) dmsDownloadVersion(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	vn, _ := strconv.Atoi(chi.URLParam(r, "v"))
	vers, err := a.dms.Versions(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var ref, name string
	for _, v := range vers {
		if v.No == vn {
			ref = v.Ref
			name = fmt.Sprintf("doc-%d-v%d", id, vn)
		}
	}
	if ref == "" {
		http.NotFound(w, r)
		return
	}
	rc, err := a.files.Open(ref)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	a.dms.LogAccess(id, a.uid(r), "DOWNLOAD", clientIPOf(r), r.UserAgent())
	w.Header().Set("Content-Disposition", "attachment; filename=\""+name+"\"")
	_, _ = io.Copy(w, rc)
}

func (a *App) dmsRestoreVersion(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	vn, _ := strconv.Atoi(chi.URLParam(r, "v"))
	vers, err := a.dms.Versions(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	var ref, sha string
	var size int64
	for _, v := range vers {
		if v.No == vn {
			ref, sha, size = v.Ref, v.Hash, v.Size
		}
	}
	if ref == "" {
		http.NotFound(w, r)
		return
	}
	if _, err := a.dms.AddVersion(a.officeOf(r), id, ref, sha, size, "restore dari v"+strconv.Itoa(vn), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.restore", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "v" + strconv.Itoa(vn)})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Versi+lama+dipulihkan+sebagai+versi+baru", http.StatusSeeOther)
}

// ---- workflow ----

func (a *App) dmsMove(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := r.FormValue("to")
	if err := a.dms.Move(a.officeOf(r), id, to, a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.workflow", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: to})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Workflow+berpindah+ke+"+to, http.StatusSeeOther)
}

// ---- relations ----

func (a *App) dmsLink(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	rid, _ := strconv.ParseInt(r.FormValue("related_id"), 10, 64)
	if err := a.dms.AddRelation(a.officeOf(r), id, r.FormValue("related_type"), rid, r.FormValue("relation")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Relasi+ditambahkan", http.StatusSeeOther)
}

// ---- favorites ----

func (a *App) dmsFavorite(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if uid := a.uid(r); uid != nil {
		_, _ = a.dms.ToggleFavorite(*uid, id, a.officeOf(r))
	}
	back := r.Referer()
	if back == "" {
		back = "/documents"
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

// ---- shares ----

func (a *App) dmsShare(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermLegalize) && !rbac.Has(mw.RoleOf(r), rbac.PermManageUsers) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	hours, _ := strconv.Atoi(r.FormValue("hours"))
	maxDl, _ := strconv.ParseInt(r.FormValue("max"), 10, 64)
	sh, err := a.dms.CreateShare(a.officeOf(r), id, hours, maxDl, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.share", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	a.render(w, r, "dms_share", "Tautan Berbagi", map[string]any{"Token": sh.Token})
}

func (a *App) dmsSharedDownload(w http.ResponseWriter, r *http.Request) {
	fid, err := a.dms.UseShare(chi.URLParam(r, "token"))
	if err != nil {
		http.Error(w, "tautan tidak valid/kedaluarsa", http.StatusGone)
		return
	}
	var ref, mime, name string
	var oid int64
	if err := a.db.QueryRow(`SELECT ref, mime, original_name, office_id FROM files WHERE id=?`, fid).
		Scan(&ref, &mime, &name, &oid); err != nil {
		http.NotFound(w, r)
		return
	}
	rc, err := a.files.Open(ref)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	a.dms.LogAccess(fid, nil, "DOWNLOAD", clientIPOf(r), "shared-link")
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Content-Disposition", "attachment; filename=\"shared-"+name+"\"")
	_, _ = io.Copy(w, rc)
}

// ---- OCR queue ----

func (a *App) dmsOCRQueue(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	jobs, _ := a.dms.OCRQueue(a.officeOf(r))
	a.render(w, r, "dms_ocr", "Antrean OCR", jobs)
}

func (a *App) dmsOCRRun(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	j, ok := a.dms.ClaimOCR()
	if !ok {
		http.Redirect(w, r, "/dms/ocr?flash=Tidak+ada+pekerjaan+antre", http.StatusSeeOther)
		return
	}
	res, err := a.uploads.RunOCRWithContext(r.Context(), j.FileID)
	if err != nil {
		_ = a.dms.FinishOCR(j.ID, false, err.Error())
		http.Redirect(w, r, "/dms/ocr?flash=Pekerjaan+gagal:+dicoba+ulang+otomatis", http.StatusSeeOther)
		return
	}
	_ = a.dms.FinishOCR(j.ID, true, "")
	_ = a.dms.IndexContent(j.FileID, res.Fields["text"])
	http.Redirect(w, r, "/dms/ocr?flash=OCR+selesai+dan+terindeks", http.StatusSeeOther)
}

// ---- search ----

func (a *App) dmsSearch(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	q := r.URL.Query().Get("q")
	canRestr := rbac.Has(mw.RoleOf(r), rbac.PermApproveDeed) || mw.RoleOf(r) == rbac.RoleAuditor || mw.RoleOf(r) == rbac.RoleSuperAdmin
	var out []map[string]any
	if strings.TrimSpace(q) != "" {
		ids, _ := a.dms.Search(a.officeOf(r), q, canRestr, 50)
		out, _ = a.dms.SearchFiles(a.officeOf(r), ids)
	}
	a.render(w, r, "dms_search", "Pencarian Dokumen", map[string]any{"Q": q, "Hits": out})
}

// ---- archive ----

func (a *App) dmsArchive(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewProtocol) {
		a.forbidden(w, r)
		return
	}
	locs, _ := a.dms.Locations(a.officeOf(r))
	type stored struct {
		ID, Name, At, Loc string
	}
	var rows []stored
	rs, _ := a.db.Query(`SELECT f.id, COALESCE(f.display_name,f.original_name), MAX(m.created_at),
		COALESCE((SELECT l.code FROM doc_archive_movements m2 LEFT JOIN doc_archive_locations l ON l.id=m2.location_id WHERE m2.file_id=f.id ORDER BY m2.id DESC LIMIT 1),'')
		FROM files f LEFT JOIN doc_archive_movements m ON m.file_id=f.id
		WHERE f.office_id=? GROUP BY f.id ORDER BY f.id DESC LIMIT 100`, a.officeOf(r))
	if rs != nil {
		defer rs.Close()
		for rs.Next() {
			var x stored
			var at sql.NullString
			_ = rs.Scan(&x.ID, &x.Name, &at, &x.Loc)
			if at.Valid {
				x.At = at.String
			}
			rows = append(rows, x)
		}
	}
	a.render(w, r, "dms_archive", "Arsip Fisik", map[string]any{"Locs": locs, "Rows": rows})
}

func (a *App) dmsArchiveLoc(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	if _, err := a.dms.CreateLocation(a.officeOf(r), r.FormValue("code"), r.FormValue("room"), r.FormValue("shelf"), r.FormValue("box")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/dms/archive?flash=Lokasi+ditambahkan", http.StatusSeeOther)
}

func (a *App) dmsArchiveMove(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var loc *int64
	if v := r.FormValue("location_id"); v != "" {
		n, _ := strconv.ParseInt(v, 10, 64)
		loc = &n
	}
	if err := a.dms.MoveArchive(a.officeOf(r), id, loc, r.FormValue("movement"), a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.archive." + strings.ToLower(r.FormValue("movement")), Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Perpindahan+arsip+dicatat", http.StatusSeeOther)
}

// ---- retention & holds ----

func (a *App) dmsRetention(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermViewReports) {
		a.forbidden(w, r)
		return
	}
	type hold struct {
		ID, FileID       int64
		Name, Reason, At string
	}
	var holds []hold
	rs, _ := a.db.Query(`SELECT h.id, h.file_id, COALESCE(f.display_name,f.original_name), h.reason, h.applied_at
		FROM doc_legal_holds h JOIN files f ON f.id=h.file_id
		WHERE h.office_id=? AND h.released_at IS NULL ORDER BY h.id DESC`, a.officeOf(r))
	if rs != nil {
		defer rs.Close()
		for rs.Next() {
			var h hold
			_ = rs.Scan(&h.ID, &h.FileID, &h.Name, &h.Reason, &h.At)
			holds = append(holds, h)
		}
	}
	exp, _ := a.dms.Expiring(a.officeOf(r))
	pols, _ := a.db.Query(`SELECT entity, retain_years, note FROM retention_policies ORDER BY entity`)
	var policies []map[string]any
	if pols != nil {
		defer pols.Close()
		for pols.Next() {
			var e string
			var y int
			var n sql.NullString
			_ = pols.Scan(&e, &y, &n)
			note := ""
			if n.Valid {
				note = n.String
			}
			policies = append(policies, map[string]any{"Entity": e, "Years": y, "Note": note})
		}
	}
	a.render(w, r, "dms_retention", "Retensi & Legal Hold", map[string]any{"Holds": holds, "Expiring": exp, "Policies": policies})
}

func (a *App) dmsHold(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.dms.ApplyHold(a.officeOf(r), id, r.FormValue("reason"), a.uid(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.hold", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Legal+hold+diterapkan", http.StatusSeeOther)
}

func (a *App) dmsReleaseHold(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
	hid, _ := strconv.ParseInt(chi.URLParam(r, "hid"), 10, 64)
	if err := a.dms.ReleaseHold(a.officeOf(r), hid); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.hold.release", Entity: "hold", EntityID: strconv.FormatInt(hid, 10)})
	http.Redirect(w, r, "/dms/retention?flash=Hold+dilepas", http.StatusSeeOther)
}

func (a *App) dmsDisposition(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err := a.dms.RequestDisposition(a.officeOf(r), id, r.FormValue("action"), a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.disposition", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: r.FormValue("action")})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Keputusan+retensi+dicatat", http.StatusSeeOther)
}

// ---- duplicates ----

func (a *App) dmsDuplicates(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	dups, _ := a.dms.DuplicateCandidates(a.officeOf(r))
	a.render(w, r, "dms_duplicates", "Deteksi Duplikat", dups)
}

// ---- admin: categories ----

func (a *App) dmsCategories(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	cats, _ := a.dms.Categories(a.officeOf(r))
	a.render(w, r, "dms_categories", "Kategori Dokumen", cats)
}

func (a *App) dmsCategoryCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	if _, err := a.dms.CreateCategory(a.officeOf(r), r.FormValue("name"), r.FormValue("description")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/dms/categories?flash=Kategori+ditambahkan", http.StatusSeeOther)
}

// ---- admin: checklist templates ----

func (a *App) dmsChecklists(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	type tpl struct {
		ID      int64
		Name    string
		Service string
		Items   string
	}
	var out []tpl
	rows, _ := a.db.Query(`SELECT t.id, t.name, t.service_code,
		(SELECT GROUP_CONCAT(label SEPARATOR ' | ') FROM doc_checklist_items WHERE template_id=t.id)
		FROM doc_checklist_templates t WHERE t.office_id=? ORDER BY t.id DESC`, a.officeOf(r))
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var t tpl
			var svc, items sql.NullString
			_ = rows.Scan(&t.ID, &t.Name, &svc, &items)
			if svc.Valid {
				t.Service = svc.String
			}
			if items.Valid {
				t.Items = items.String
			}
			out = append(out, t)
		}
	}
	svcs, _ := a.reqs.Services()
	a.render(w, r, "dms_checklists", "Template Checklist", map[string]any{"Tpls": out, "Services": svcs})
}

func (a *App) dmsChecklistCreate(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	_ = r.ParseForm()
	var items []string
	for _, ln := range strings.Split(r.FormValue("items"), "\n") {
		if s := strings.TrimSpace(ln); s != "" {
			items = append(items, s)
		}
	}
	if _, err := a.dms.CreateChecklistTemplate(a.officeOf(r), r.FormValue("service_code"), r.FormValue("name"), items); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/dms/checklists?flash=Template+checklist+disimpan", http.StatusSeeOther)
}

func (a *App) dmsChecklistSet(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	itemID, _ := strconv.ParseInt(r.FormValue("item_id"), 10, 64)
	if err := a.dms.SetChecklistItem(a.officeOf(r), reqID, itemID, r.FormValue("status"), a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Checklist+diperbarui", http.StatusSeeOther)
}
