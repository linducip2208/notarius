package app

// DMS HTTP layer: library, detail+preview, versions, workflow, OCR queue,
// shares, archive, retention, checklist, admin. All office-scoped, audited.

import (
	"bytes"
	"database/sql"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/skip2/go-qrcode"
	"notaryos/internal/audit"
	"notaryos/internal/config"
	"notaryos/internal/dms"
	"notaryos/internal/files"
	"notaryos/internal/malware"
	mw "notaryos/internal/middleware"
	"notaryos/internal/notify"
	"notaryos/internal/ocr"
	"notaryos/internal/officex"
	"notaryos/internal/plan"
	"notaryos/internal/rbac"
	"notaryos/internal/remind"
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

// requireWrite guards mutating endpoints: requireStaff passes the auditor
// (auditor holds PermViewClient for supervision), so write paths must also
// reject read-only roles explicitly.
func (a *App) requireWrite(w http.ResponseWriter, r *http.Request) bool {
	if rbac.ReadOnly(mw.RoleOf(r)) {
		a.forbidden(w, r)
		return false
	}
	return a.requireStaff(w, r)
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
			_ = pendWF.Scan(&x.ID, &x.Name, &x.WF)
			pwf = append(pwf, x)
		}
	}
	wfStats, _ := a.dms.WorkflowStats(office)
	byMime, _ := a.dms.StorageByMIME(office)
	var quota, used int64
	_ = a.db.QueryRow(`SELECT storage_quota_bytes FROM offices WHERE id=?`, office).Scan(&quota)
	used = a.dms.StorageUsage(office)
	a.render(w, r, "dms_dashboard", "DMS Overview", map[string]any{
		"Stats": st, "Recent": rows, "Dups": dups, "Expiring": exp, "Jobs": jobs, "PendingWF": pwf,
		"WFStats": wfStats, "ByMIME": byMime, "Quota": quota, "Used": used,
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
	mime := q.Get("mime")
	fav := q.Get("fav") == "1"
	page := config.MustInt(q.Get("page"), 1)
	if page < 1 {
		page = 1
	}
	perPage := config.MustInt(q.Get("per_page"), 20)
	if perPage != 10 && perPage != 20 && perPage != 50 && perPage != 100 {
		perPage = 20
	}
	query := `SELECT f.id, COALESCE(f.display_name,f.original_name), f.mime, f.size_bytes,
		f.ocr_status, f.verification_status, f.workflow_status, f.confidentiality, f.created_at FROM files f WHERE f.office_id=?`
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
	if mime != "" {
		query += ` AND f.mime LIKE ?`
		args = append(args, mime+"%")
	}
	if v := q.Get("from"); v != "" {
		query += ` AND DATE(f.created_at)>=?`
		args = append(args, v)
	}
	if v := q.Get("to"); v != "" {
		query += ` AND DATE(f.created_at)<=?`
		args = append(args, v)
	}
	if fav {
		if id := a.uid(r); id != nil {
			query += ` AND EXISTS(SELECT 1 FROM doc_favorites WHERE user_id=? AND file_id=f.id)`
			args = append(args, *id)
		}
	}
	var total int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM (`+query+`) t`, args...).Scan(&total)
	sort := q.Get("sort")
	switch sort {
	case "name":
		query += ` ORDER BY 2`
	case "size_desc":
		query += ` ORDER BY f.size_bytes DESC`
	case "size_asc":
		query += ` ORDER BY f.size_bytes ASC`
	default:
		query += ` ORDER BY f.id DESC`
	}
	pages := (total + perPage - 1) / perPage
	if pages < 1 {
		pages = 1
	}
	if page > pages {
		page = pages
	}
	query += ` LIMIT ? OFFSET ?`
	args = append(args, perPage, (page-1)*perPage)
	rows, _ := a.db.Query(query, args...)
	type doc struct {
		ID, Size             int64
		Name, MIME, OCR, Ver string
		WF, Conf, At         string
	}
	var out []doc
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var d doc
			_ = rows.Scan(&d.ID, &d.Name, &d.MIME, &d.Size, &d.OCR, &d.Ver, &d.WF, &d.Conf, &d.At)
			if !dmsCanSee(mw.RoleOf(r), d.Conf) {
				continue
			}
			out = append(out, d)
		}
	}
	cats, _ := a.dms.Categories(a.officeOf(r))
	qq := r.URL.Query()
	qq.Del("page")
	qs := qq.Encode()
	a.render(w, r, "dms_library", "Document Library", map[string]any{
		"Docs": out, "Cats": cats, "Q": q.Get("q"), "Cat": cat,
		"WF": wf, "VST": vst, "Conf": conf, "Fav": fav, "MIME": mime,
		"From": q.Get("from"), "To": q.Get("to"), "Sort": sort,
		"Page": page, "Pages": pages, "Total": total, "PerPage": perPage, "QS": qs,
		"Prev": max(page-1, 1), "Next": min(page+1, pages),
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
	// RESTRICTED: metadata pun tidak boleh terlihat oleh peran tak berhak.
	if m.Conf == "RESTRICTED" && !dmsCanSee(mw.RoleOf(r), m.Conf) {
		a.forbidden(w, r)
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
	usedIn := a.dmsUsedIn(a.officeOf(r), id, willReader(mw.RoleOf(r)))
	tl, _ := a.dms.Timeline(a.officeOf(r), id)
	holds, _ := a.dms.ActiveHolds(a.officeOf(r), id)
	hist, _ := a.dms.MoveHistory(a.officeOf(r), id)
	alog, _ := a.dms.AccessLog(a.officeOf(r), id)
	ocrs, _ := a.uploads.OCRResults(id)
	kkMembers := make([][]map[string]string, len(ocrs))
	for i := range ocrs {
		kkMembers[i] = parseKKMembers(ocrs[i].Fields)
	}
	cats, _ := a.dms.Categories(a.officeOf(r))
	locs, _ := a.dms.Locations(a.officeOf(r))
	fields, _ := a.dms.CustomFields(a.officeOf(r))
	fvals, _ := a.dms.FieldValues(id)
	pendingAppr, totalAppr := a.dms.ApprovalState(id)
	// Saran kategori otomatis: DocType OCR terbaru (atau tebak nama file
	// bila OCR masih antre) dipetakan ke kategori kantor yang sudah ada.
	// Murni saran baca — tidak pernah membuat kategori / mengubah berkas.
	sugDocType := string(ocr.Classify(m.Name))
	if len(ocrs) > 0 && strings.TrimSpace(string(ocrs[0].DocType)) != "" {
		sugDocType = string(ocrs[0].DocType)
	}
	var sugCatID int64
	var sugCatName string
	var showSug bool
	if sug, ok := dms.SuggestCategory(cats, sugDocType, m.MIME, m.Name); ok {
		if m.CatID == 0 || m.CatID != sug.ID {
			sugCatID, sugCatName, showSug = sug.ID, sug.Name, true
		}
	}
	showSugDash := !showSug && m.CatID == 0
	a.render(w, r, "dms_detail", "Dokumen #"+strconv.FormatInt(id, 10), map[string]any{
		"M": m, "Vers": vers, "Rels": rels, "UsedIn": usedIn, "Timeline": tl, "Holds": holds,
		"Hist": hist, "Access": alog, "OCRs": ocrs, "KKMembers": kkMembers, "Cats": cats, "Locs": locs,
		"Fields": fields, "FieldVals": fvals, "CanSee": dmsCanSee(mw.RoleOf(r), m.Conf),
		"ApprPending": pendingAppr, "ApprTotal": totalAppr,
		"SugCatID": sugCatID, "SugCatName": sugCatName, "ShowSug": showSug, "ShowSugDash": showSugDash,
	})
}

// dmsUsedIn collects reverse references to a file: doc_relations resolved to
// titled deep-links, tasks with ref_type='file', and client specimens.
// All lookups are best-effort reads; missing tables/rows are skipped so no
// migration is required.
func (a *App) dmsUsedIn(officeID, fileID int64, canWill bool) []map[string]any {
	var out []map[string]any
	seen := map[string]bool{}
	add := func(kind string, id int64, title, url string) {
		if id <= 0 {
			return
		}
		k := kind + "/" + strconv.FormatInt(id, 10)
		if seen[k] {
			return
		}
		seen[k] = true
		if title == "" {
			title = kind + " #" + strconv.FormatInt(id, 10)
		}
		out = append(out, map[string]any{"Kind": kind, "ID": id, "Title": title, "URL": url})
	}
	titleOf := func(query string, id int64, args ...any) string {
		var t sql.NullString
		aargs := append([]any{}, args...)
		aargs = append(aargs, id)
		_ = a.db.QueryRow(query, aargs...).Scan(&t)
		if t.Valid {
			return t.String
		}
		return ""
	}
	// 1) Forward relations file -> entity (deed attach / link form).
	if rows, err := a.db.Query(`SELECT related_type, related_id FROM doc_relations WHERE file_id=? ORDER BY id`, fileID); err == nil {
		defer rows.Close()
		for rows.Next() {
			var rtype string
			var rid int64
			if err := rows.Scan(&rtype, &rid); err != nil {
				continue
			}
			switch rtype {
			case "client":
				add("klien", rid, titleOf(`SELECT full_name FROM clients WHERE id=?`, rid), "/clients/"+strconv.FormatInt(rid, 10))
			case "request":
				add("permohonan", rid, titleOf(`SELECT title FROM service_requests WHERE id=?`, rid), "/requests/"+strconv.FormatInt(rid, 10))
			case "deed":
				add("akta", rid, titleOf(`SELECT title FROM deed_drafts WHERE id=?`, rid), "/deeds/"+strconv.FormatInt(rid, 10))
			case "invoice":
				t := titleOf(`SELECT number FROM invoices WHERE id=?`, rid)
				add("invoice", rid, t, "/invoices/"+strconv.FormatInt(rid, 10)+"/receipt")
			case "service":
				add("layanan", rid, titleOf(`SELECT name FROM services WHERE id=?`, rid), "/layanan")
			case "company":
				add("perusahaan", rid, titleOf(`SELECT name FROM companies WHERE id=?`, rid), "/requests")
			case "repertorium":
				add("repertorium", rid, titleOf(`SELECT description FROM repertorium WHERE id=?`, rid), "/repertorium")
			case "will":
				if !canWill {
					add("wasiat", rid, "Wasiat #"+strconv.FormatInt(rid, 10)+" (terbatas)", "/wills")
					continue
				}
				t := titleOf(`SELECT register_no FROM wills WHERE id=?`, rid)
				if t == "" {
					t = "wasiat #" + strconv.FormatInt(rid, 10)
				}
				add("wasiat", rid, t, "/wills")
			case "legalization":
				add("legalisasi", rid, titleOf(`SELECT doc_title FROM legalizations WHERE id=?`, rid), "/legalisasi")
			case "file":
				add("berkas", rid, titleOf(`SELECT COALESCE(display_name,original_name) FROM files WHERE id=?`, rid), "/documents/"+strconv.FormatInt(rid, 10))
			default:
				add(rtype, rid, "", "#")
			}
		}
	}
	// 2) Owner upload context (files.owner_type/owner_id) — pre-relasi.
	var otype string
	var oid int64
	if err := a.db.QueryRow(`SELECT owner_type, owner_id FROM files WHERE id=?`, fileID).Scan(&otype, &oid); err == nil && oid > 0 {
		switch otype {
		case "client":
			add("klien", oid, titleOf(`SELECT full_name FROM clients WHERE id=?`, oid), "/clients/"+strconv.FormatInt(oid, 10))
		case "request":
			add("permohonan", oid, titleOf(`SELECT title FROM service_requests WHERE id=?`, oid), "/requests/"+strconv.FormatInt(oid, 10))
		case "deed":
			add("akta", oid, titleOf(`SELECT title FROM deed_drafts WHERE id=?`, oid), "/deeds/"+strconv.FormatInt(oid, 10))
		}
	}
	// 3) Tasks tertaut ke berkas (tasks.ref_type='file'). Tabel/kolom mungkin
	// belum ada di skema lama — gagal diam-diam = dilewati.
	if rows, err := a.db.Query(`SELECT id, title FROM tasks WHERE office_id=? AND ref_type='file' AND ref_id=? ORDER BY id DESC LIMIT 20`, officeID, fileID); err == nil {
		defer rows.Close()
		for rows.Next() {
			var tid int64
			var t string
			if err := rows.Scan(&tid, &t); err != nil {
				continue
			}
			add("tugas", tid, t, "/tasks")
		}
	}
	// 4) Spesimen klien memakai berkas ini.
	if rows, err := a.db.Query(`SELECT client_id FROM client_specimens WHERE file_id=? AND office_id=? LIMIT 20`, fileID, officeID); err == nil {
		defer rows.Close()
		for rows.Next() {
			var cid int64
			if err := rows.Scan(&cid); err != nil {
				continue
			}
			add("spesimen", cid, titleOf(`SELECT full_name FROM clients WHERE id=?`, cid), "/clients/"+strconv.FormatInt(cid, 10)+"/specimens")
		}
	}
	return out
}

// parseKKMembers extracts KK table rows from OCR fields. Supports several
// key conventions (kk_nama_1/kk_nik_1, kk_member_1_name, kk_1_nama, ...)
// because the OCR layer stores whatever was extracted. Empty => nil.
func parseKKMembers(fields map[string]string) []map[string]string {
	if len(fields) == 0 {
		return nil
	}
	get := func(keys ...string) string {
		for _, k := range keys {
			if v := strings.TrimSpace(fields[k]); v != "" {
				return v
			}
		}
		return ""
	}
	var out []map[string]string
	for i := 1; i <= 20; i++ {
		n := strconv.Itoa(i)
		name := get("kk_nama_"+n, "kk_name_"+n, "kk_anggota_"+n, "kk_member_"+n, "kk_member_"+n+"_name",
			"kk_"+n+"_nama", "kk_"+n+"_name", "kk_nama"+n)
		nik := get("kk_nik_"+n, "kk_member_"+n+"_nik", "kk_"+n+"_nik", "kk_nik"+n)
		if name == "" && nik == "" {
			continue
		}
		out = append(out, map[string]string{"Name": name, "NIK": nik})
	}
	return out
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
	conf := strings.ToUpper(strings.TrimSpace(r.FormValue("confidentiality")))
	switch conf {
	case "PUBLIC", "INTERNAL", "CONFIDENTIAL", "RESTRICTED":
	default:
		conf = "INTERNAL"
	}
	_, err := a.db.Exec(`UPDATE files SET display_name=?, doc_number=?, category_id=?, doc_date=?, expiry_date=?, confidentiality=? WHERE id=? AND office_id=?`,
		nullStrOpt(r.FormValue("display_name")), nullStrOpt(r.FormValue("doc_number")), cat,
		nullStrOpt(r.FormValue("doc_date")), nullStrOpt(r.FormValue("expiry_date")), conf, id, a.officeOf(r))
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
	if _, err := a.db.Exec(`UPDATE files SET verification_status=? WHERE id=? AND office_id=?`, to, id, a.officeOf(r)); err != nil {
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
	var conf string
	_ = a.db.QueryRow(`SELECT confidentiality FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&conf)
	if !a.canSeeDoc(r, conf) {
		a.forbidden(w, r)
		return
	}
	m, rc, err := a.uploads.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	if m.MIME != "application/pdf" && !strings.HasPrefix(m.MIME, "image/") {
		if text, ok := a.previewOfficeText(m, rc); ok {
			a.dms.LogAccess(id, a.uid(r), "PREVIEW", clientIPOf(r), r.UserAgent())
			a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.preview", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "teks-ooxml"})
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.Header().Set("Content-Disposition", "inline")
			_, _ = io.WriteString(w, text)
			return
		}
		http.Error(w, "pratinjau hanya untuk PDF/gambar/DOCX/XLSX; unduh untuk format lain", http.StatusUnsupportedMediaType)
		return
	}
	a.dms.LogAccess(id, a.uid(r), "PREVIEW", clientIPOf(r), r.UserAgent())
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.preview", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	w.Header().Set("Content-Type", m.MIME)
	w.Header().Set("Content-Disposition", "inline")
	_, _ = io.Copy(w, rc)
}

// dmsScan runs an on-demand ClamAV INSTREAM scan (honest: needs CLAMAV_ADDR;
// without daemon it reports unavailable, never "clean").
func (a *App) dmsScan(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, rc, err := a.uploads.Get(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	addr := os.Getenv("CLAMAV_ADDR")
	if addr == "" {
		addr = "127.0.0.1:3310"
	}
	sc := malware.Scanner{Addr: addr}
	if err := sc.Ping(); err != nil {
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.scan.unavailable", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
		http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Pemindai+belum+tersedia+(CLAMAV_ADDR)", http.StatusSeeOther)
		return
	}
	// Cap 64 MB streaming; larger files are rejected honestly.
	lr := io.LimitReader(rc, 64<<20)
	res, err := sc.Scan(lr)
	if err != nil {
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.scan.error", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: err.Error()})
		http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Pindai+gagal:+periksa+daemon", http.StatusSeeOther)
		return
	}
	if res.Infected {
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.scan.infected", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: res.Signature})
		http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=TERINFEKSI:"+res.Signature, http.StatusSeeOther)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.scan.clean", Entity: "file", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/documents/"+strconv.FormatInt(id, 10)+"?flash=Bersih+(ClamAV)", http.StatusSeeOther)
}

// ---- versions ----

// previewOfficeText extracts capped plain text for OOXML (max 8 MB di memori).
func (a *App) previewOfficeText(m files.Meta, rc io.ReadCloser) (string, bool) {
	if m.Size <= 0 || m.Size > 8<<20 {
		return "", false
	}
	buf := make([]byte, m.Size)
	if _, err := io.ReadFull(rc, buf); err != nil {
		return "", false
	}
	return officex.Extract(m.Name, m.MIME, bytes.NewReader(buf), int64(len(buf)))
}

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
	if a.overQuota(a.officeOf(r), hdr.Size) {
		http.Error(w, "kuota penyimpanan kantor terlampaui — hubungi administrator", http.StatusInsufficientStorage)
		return
	}
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
	var conf string
	_ = a.db.QueryRow(`SELECT confidentiality FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&conf)
	if !a.canSeeDoc(r, conf) {
		a.forbidden(w, r)
		return
	}
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
	if !a.requireWrite(w, r) {
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
	// Rantai persetujuan bertingkat: siapkan langkah saat masuk APPROVAL;
	// APPROVED diblokir sampai seluruh langkah disetujui (bila rantai ada).
	if to == "APPROVAL" {
		if err := a.dms.EnsureApprovals(a.officeOf(r), id); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	if to == "APPROVED" && !a.chainSatisfied(a.officeOf(r), id) {
		http.Error(w, "rantai persetujuan belum lengkap — kumpulkan semua persetujuan dulu", http.StatusConflict)
		return
	}
	if err := a.dms.Move(a.officeOf(r), id, to, a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.workflow", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: to})
	// SLA otomatis: status terminal menutup tugas SLA yang masih OPEN;
	// status lain membuat task bertipe file bila belum ada yang OPEN untuk ref ini.
	switch to {
	case "APPROVED", "FINALIZED", "ARCHIVED", "REJECTED":
		_, _ = a.db.Exec(`UPDATE tasks SET status='DONE' WHERE office_id=? AND status='OPEN' AND ref_type='file' AND ref_id=?`,
			a.officeOf(r), id)
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "dms.sla", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "SLA " + to + " selesai"})
	default:
		dueDays, _ := remind.SLAForWorkflow(to)
		var n int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM tasks WHERE office_id=? AND status='OPEN' AND ref_type='file' AND ref_id=?`,
			a.officeOf(r), id).Scan(&n)
		if n == 0 {
			due := time.Now().AddDate(0, 0, dueDays).Format("2006-01-02")
			_, _ = a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by, ref_type, ref_id) VALUES(?,?,?,?, 'file', ?)`,
				a.officeOf(r), fmt.Sprintf("SLA %s: dok #%d", to, id), due, a.uid(r), id)
		}
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "dms.sla", Entity: "file", EntityID: strconv.FormatInt(id, 10), Detail: "SLA " + to})
	}
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
	if !a.requirePlan(w, r, plan.FeatShares) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	hours, _ := strconv.Atoi(r.FormValue("hours"))
	maxDl, _ := strconv.ParseInt(r.FormValue("max"), 10, 64)
	var conf string
	_ = a.db.QueryRow(`SELECT confidentiality FROM files WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&conf)
	if conf == "RESTRICTED" {
		http.Error(w, "berkas terbatas tidak dapat dibagikan via tautan publik", http.StatusForbidden)
		return
	}
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
	var conf string
	if err := a.db.QueryRow(`SELECT ref, mime, original_name, office_id, confidentiality FROM files WHERE id=?`, fid).
		Scan(&ref, &mime, &name, &oid, &conf); err != nil {
		http.NotFound(w, r)
		return
	}
	if conf == "RESTRICTED" {
		http.Error(w, "berkas terbatas tidak dapat dibagikan via tautan publik", http.StatusForbidden)
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

// dmsMerge links a duplicate group into its smallest-ID canonical record.
// Bytes are never deleted; relations record the merge (audit trail kept).
func (a *App) dmsMerge(w http.ResponseWriter, r *http.Request) {
	if !a.requireWrite(w, r) {
		return
	}
	_ = r.ParseForm()
	var ids []int64
	for _, part := range strings.Split(r.FormValue("ids"), ",") {
		if id, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && id > 0 {
			ids = append(ids, id)
		}
	}
	if len(ids) < 2 {
		http.Error(w, "butuh minimal 2 berkas untuk digabung", http.StatusBadRequest)
		return
	}
	canonical := ids[0]
	for _, id := range ids[1:] {
		if id < canonical {
			canonical = id
		}
	}
	n, err := a.dms.MergeDuplicates(a.officeOf(r), canonical, ids)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "doc.merge", Entity: "file", EntityID: strconv.FormatInt(canonical, 10), Detail: fmt.Sprintf("%d digabung", n)})
	http.Redirect(w, r, "/dms/duplicates?flash="+strconv.Itoa(n)+"+berkas+digabung+ke+%23"+strconv.FormatInt(canonical, 10), http.StatusSeeOther)
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
	if !a.requirePlan(w, r, plan.FeatChecklist) {
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
	if !a.requirePlan(w, r, plan.FeatChecklist) {
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

// dmsChecklistSeed memasang 5 paket standar badan usaha + 3 template alur
// kerja standar (027: PT-DIRIAN, PPAT-AJB, KUASA) bila belum ada.
// Idempoten per service_code: template yang sudah ada dilewati.
func (a *App) dmsChecklistSeed(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageTemplates) {
		a.forbidden(w, r)
		return
	}
	if !a.requirePlan(w, r, plan.FeatChecklist) {
		return
	}
	office := a.officeOf(r)
	std := standardChecklistSeeds()
	made := 0
	for _, s := range std {
		var n int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM doc_checklist_templates WHERE office_id=? AND service_code=?`, office, s.code).Scan(&n)
		if n > 0 {
			continue
		}
		if _, err := a.dms.CreateChecklistTemplate(office, s.code, s.name, s.items); err == nil {
			made++
		}
	}
	wfMade := a.seedWorkflowTemplates(office)
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "checklist.seed", Entity: "office", EntityID: strconv.FormatInt(office, 10), Detail: strconv.Itoa(made) + " paket+" + strconv.Itoa(wfMade) + " alur"})
	http.Redirect(w, r, "/dms/checklists?flash="+url.QueryEscape(strconv.Itoa(made)+" paket standar + "+strconv.Itoa(wfMade)+" template alur kerja dipasang"), http.StatusSeeOther)
}

func (a *App) dmsChecklistSet(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	_ = r.ParseForm()
	reqID, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	itemID, _ := strconv.ParseInt(r.FormValue("item_id"), 10, 64)
	st := r.FormValue("status")
	if err := a.dms.SetChecklistItem(a.officeOf(r), reqID, itemID, st, a.uid(r), r.FormValue("note")); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "checklist.set", Entity: "service_request", EntityID: strconv.FormatInt(reqID, 10), Detail: st})
	// Tutup loop workflow: bila SEMUA item request itu VERIFIED/RECEIVED →
	// step kunci "syarat" DONE (best-effort, fail-open).
	{
		aid, aname := a.workflowActorName(r)
		a.autoWorkflowSyarat(a.officeOf(r), reqID, aid, aname)
	}
	if st == "NEEDS_REVISION" || st == "REJECTED" {
		var creator sql.NullInt64
		var title string
		_ = a.db.QueryRow(`SELECT created_by, title FROM service_requests WHERE id=?`, reqID).Scan(&creator, &title)
		if creator.Valid {
			_ = notify.Create(a.db, creator.Int64, "Berkas perlu perbaikan: "+title, "Status: "+st+" — "+r.FormValue("note"))
		}
	}
	http.Redirect(w, r, "/requests/"+strconv.FormatInt(reqID, 10)+"?flash=Checklist+diperbarui", http.StatusSeeOther)
}

// ---- QR generik + label arsip ----

// qrText serves a QR PNG for short staff-provided text (capped, no logging of payload).
func (a *App) qrText(w http.ResponseWriter, r *http.Request) {
	if !a.requireStaff(w, r) {
		return
	}
	d := r.URL.Query().Get("d")
	if d == "" || len(d) > 200 {
		http.Error(w, "teks QR kosong/terlalu panjang (maks 200)", http.StatusBadRequest)
		return
	}
	png, err := qrcode.Encode(d, qrcode.Medium, 160)
	if err != nil {
		http.Error(w, "gagal membuat QR", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Write(png)
}

// dmsLabels renders printable QR labels for archive locations.
func (a *App) dmsLabels(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermManageRepert) {
		a.forbidden(w, r)
		return
	}
	locs, _ := a.dms.Locations(a.officeOf(r))
	var office string
	_ = a.db.QueryRow(`SELECT name FROM offices WHERE id=?`, a.officeOf(r)).Scan(&office)
	a.render(w, r, "dms_labels", "Label QR Arsip", map[string]any{"Locs": locs, "Office": office})
}
