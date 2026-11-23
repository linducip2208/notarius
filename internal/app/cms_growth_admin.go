package app

// CMS growth (admin): inquiries, bookings+slots, team/gallery/testimonials,
// content versions + rollback, quality dashboard, pageview stats, ZIP export.

import (
	"archive/zip"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/clients"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
	"notaryos/internal/validate"
)

// ---- inquiries ----

func (a *App) cmsInquiries(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebInquiries) {
		return
	}
	st := r.URL.Query().Get("st")
	q := `SELECT id, name, contact, service, message, status, created_at FROM website_inquiries WHERE office_id=?`
	args := []any{a.officeOf(r)}
	if st != "" {
		q += ` AND status=?`
		args = append(args, st)
	}
	q += ` ORDER BY id DESC LIMIT 200`
	rows, _ := a.db.Query(q, args...)
	type inq struct {
		ID                             int64
		Name, Contact, Service, Msg, St string
		At                             string
	}
	var out []inq
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x inq
			_ = rows.Scan(&x.ID, &x.Name, &x.Contact, &x.Service, &x.Msg, &x.St, &x.At)
			out = append(out, x)
		}
	}
	var nb int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_inquiries WHERE office_id=? AND status='BARU'`, a.officeOf(r)).Scan(&nb)
	type svc struct {
		ID   int64
		Name string
	}
	var svcs []svc
	if srows, err := a.db.Query(`SELECT id, name FROM services WHERE is_active=1 ORDER BY name`); err == nil && srows != nil {
		defer srows.Close()
		for srows.Next() {
			var x svc
			_ = srows.Scan(&x.ID, &x.Name)
			svcs = append(svcs, x)
		}
	}
	a.render(w, r, "cms_inquiries", "Konsultasi Masuk", map[string]any{"Rows": out, "St": st, "NewCount": nb, "Svcs": svcs})
}

func (a *App) cmsInquiryStatus(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebInquiries) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := strings.ToUpper(strings.TrimSpace(r.FormValue("to")))
	switch to {
	case "BARU", "DIPROSES", "SELESAI", "SPAM":
	default:
		http.Error(w, "status tidak dikenal", http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`UPDATE website_inquiries SET status=? WHERE id=? AND office_id=?`, to, id, a.officeOf(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.inquiry.status", Entity: "inquiry", EntityID: strconv.FormatInt(id, 10), Detail: to})
	http.Redirect(w, r, "/cms/inquiries?flash=Status+diperbarui", http.StatusSeeOther)
}

// ---- slots & bookings ----

func (a *App) cmsSlots(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	type s struct {
		ID           int64
		Weekday      int
		Day          string
		Start, End   string
		Quota, Active int
	}
	dayName := map[int]string{0: "Minggu", 1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu"}
	var out []s
	rows, _ := a.db.Query(`SELECT id, weekday, DATE_FORMAT(start_time,'%H:%i'), DATE_FORMAT(end_time,'%H:%i'), quota, is_active FROM website_slots WHERE office_id=? ORDER BY weekday, start_time`, a.officeOf(r))
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x s
			_ = rows.Scan(&x.ID, &x.Weekday, &x.Start, &x.End, &x.Quota, &x.Active)
			x.Day = dayName[x.Weekday%7]
			out = append(out, x)
		}
	}
	a.render(w, r, "cms_slots", "Slot Booking", map[string]any{"Rows": out})
}

func (a *App) cmsSlotSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	_ = r.ParseForm()
	wd, _ := strconv.Atoi(r.FormValue("weekday"))
	start, end := strings.TrimSpace(r.FormValue("start")), strings.TrimSpace(r.FormValue("end"))
	quota, _ := strconv.Atoi(r.FormValue("quota"))
	if wd < 0 || wd > 6 || start == "" || end == "" || start >= end {
		http.Error(w, "hari/jam tidak valid", http.StatusBadRequest)
		return
	}
	if quota < 1 || quota > 50 {
		quota = 4
	}
	if _, err := a.db.Exec(`INSERT INTO website_slots(office_id, weekday, start_time, end_time, quota) VALUES(?,?,?,?,?)`,
		a.officeOf(r), wd, start, end, quota); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.slot.create", Entity: "slot", EntityID: "new"})
	http.Redirect(w, r, "/cms/slots?flash=Slot+ditambah", http.StatusSeeOther)
}

func (a *App) cmsSlotDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`DELETE FROM website_slots WHERE id=? AND office_id=?`, id, a.officeOf(r))
	http.Redirect(w, r, "/cms/slots?flash=Slot+dihapus", http.StatusSeeOther)
}

func (a *App) cmsBookings(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebInquiries) {
		return
	}
	st := r.URL.Query().Get("st")
	q := `SELECT id, DATE_FORMAT(book_date,'%Y-%m-%d'), DATE_FORMAT(start_time,'%H:%i'), name, contact, purpose, status, COALESCE(appointment_id,0) FROM website_bookings WHERE office_id=?`
	args := []any{a.officeOf(r)}
	if st != "" {
		q += ` AND status=?`
		args = append(args, st)
	}
	q += ` ORDER BY book_date DESC, start_time DESC LIMIT 200`
	rows, _ := a.db.Query(q, args...)
	type b struct {
		ID                       int64
		Date, Start, Name         string
		Contact, Purpose, Status  string
		Appt                     int64
	}
	var out []b
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x b
			_ = rows.Scan(&x.ID, &x.Date, &x.Start, &x.Name, &x.Contact, &x.Purpose, &x.Status, &x.Appt)
			out = append(out, x)
		}
	}
	var nw int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_bookings WHERE office_id=? AND status='MENUNGGU'`, a.officeOf(r)).Scan(&nw)
	a.render(w, r, "cms_bookings", "Booking Jadwal", map[string]any{"Rows": out, "St": st, "WaitCount": nw})
}

func (a *App) cmsBookingDecide(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebInquiries) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := strings.ToUpper(strings.TrimSpace(r.FormValue("to")))
	switch to {
	case "DISETUJUI", "DITOLAK", "BATAL":
	default:
		http.Error(w, "keputusan tidak dikenal", http.StatusBadRequest)
		return
	}
	var bdate, bstart, bname, bcontact, bpurpose, bstatus string
	var bappt sql.NullInt64
	if err := a.db.QueryRow(`SELECT DATE_FORMAT(book_date,'%Y-%m-%d'), DATE_FORMAT(start_time,'%H:%i'), name, contact, purpose, status, appointment_id FROM website_bookings WHERE id=? AND office_id=?`,
		id, a.officeOf(r)).Scan(&bdate, &bstart, &bname, &bcontact, &bpurpose, &bstatus, &bappt); err != nil {
		http.NotFound(w, r)
		return
	}
	if bstatus != "MENUNGGU" && to == "DISETUJUI" {
		http.Error(w, "hanya booking MENUNGGU yang bisa disetujui", http.StatusConflict)
		return
	}
	if to == "DISETUJUI" {
		// Buat klien tamu + appointment nyata via jalur resmi.
		// Booking mensyaratkan nomor HP valid (diverifikasi saat pengajuan).
		if validate.Phone(bcontact) != nil {
			http.Error(w, "kontak booking bukan nomor HP valid — tolak dan minta klien mengulang via konsultasi", http.StatusBadRequest)
			return
		}
		cid, err := a.clients.Create(clients.Client{OfficeID: a.officeOf(r), Kind: "individual", FullName: bname, Phone: bcontact})
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		at, err := time.Parse("2006-01-02 15:04", bdate+" "+bstart)
		if err != nil {
			http.Error(w, "jadwal booking rusak", http.StatusBadRequest)
			return
		}
		apptID, err := a.reqs.Schedule(a.officeOf(r), cid, nil, at, "Booking situs: "+bpurpose)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		_, _ = a.db.Exec(`UPDATE website_bookings SET status='DISETUJUI', appointment_id=? WHERE id=?`, apptID, id)
		// Tugas persiapan office-scoped agar kunjungan tidak menggantung (best-effort:
		// kegagalan insert tidak menggagalkan approval; tanpa ubah skema/status flow).
		need := strings.TrimSpace(bpurpose)
		if need == "" {
			need = "umum"
		}
		taskTitle := fmt.Sprintf("Siapkan kunjungan booking: %s — %s (%s %s)", need, bname, bdate, bstart)
		_, _ = a.db.Exec(`INSERT INTO tasks(office_id, title, due_date, created_by) VALUES(?,?,?,?)`,
			a.officeOf(r), taskTitle, bdate, a.uid(r))
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.booking.approve", Entity: "booking", EntityID: strconv.FormatInt(id, 10), Detail: fmt.Sprintf("appt#%d", apptID)})
	} else {
		_, _ = a.db.Exec(`UPDATE website_bookings SET status=? WHERE id=?`, to, id)
		a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.booking.decide", Entity: "booking", EntityID: strconv.FormatInt(id, 10), Detail: to})
	}
	http.Redirect(w, r, "/cms/bookings?flash=Booking+"+to, http.StatusSeeOther)
}

// ---- team / gallery / testimonials ----

func (a *App) cmsTeam(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	rows, _ := a.db.Query(`SELECT id, name, title, photo_media_id, position, visible FROM website_team WHERE office_id=? ORDER BY position, id`, a.officeOf(r))
	type m struct {
		ID             int64
		Name, Title    string
		Photo          int64
		Pos, Vis       int
	}
	var out []m
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x m
			var ph sql.NullInt64
			_ = rows.Scan(&x.ID, &x.Name, &x.Title, &ph, &x.Pos, &x.Vis)
			if ph.Valid {
				x.Photo = ph.Int64
			}
			out = append(out, x)
		}
	}
	a.render(w, r, "cms_team", "Tim Kantor", map[string]any{"Rows": out})
}

func (a *App) cmsTeamSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" {
		http.Error(w, "nama wajib diisi", http.StatusBadRequest)
		return
	}
	pos, _ := strconv.Atoi(r.FormValue("position"))
	var photo any
	if v := strings.TrimSpace(r.FormValue("photo_media_id")); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			photo = n
		}
	}
	_, err := a.db.Exec(`INSERT INTO website_team(office_id, name, title, photo_media_id, position, visible) VALUES(?,?,?,?,?,?)`,
		a.officeOf(r), name, r.FormValue("title"), photo, pos, boolToInt(r.FormValue("visible") != "0"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/team?flash=Anggota+ditambah", http.StatusSeeOther)
}

func (a *App) cmsTeamDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`DELETE FROM website_team WHERE id=? AND office_id=?`, id, a.officeOf(r))
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/team?flash=Dihapus", http.StatusSeeOther)
}

func (a *App) cmsGallery(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	rows, _ := a.db.Query(`SELECT id, media_id, caption, position FROM website_gallery WHERE office_id=? ORDER BY position, id`, a.officeOf(r))
	type g struct {
		ID           int64
		Media        int64
		Caption      string
		Pos          int
	}
	var out []g
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x g
			_ = rows.Scan(&x.ID, &x.Media, &x.Caption, &x.Pos)
			out = append(out, x)
		}
	}
	a.render(w, r, "cms_gallery", "Galeri", map[string]any{"Rows": out})
}

func (a *App) cmsGallerySave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	_ = r.ParseForm()
	media, _ := strconv.ParseInt(r.FormValue("media_id"), 10, 64)
	if media <= 0 {
		http.Error(w, "ID media wajib (lihat ID di Media Library)", http.StatusBadRequest)
		return
	}
	pos, _ := strconv.Atoi(r.FormValue("position"))
	_, err := a.db.Exec(`INSERT INTO website_gallery(office_id, media_id, caption, position) VALUES(?,?,?,?)`,
		a.officeOf(r), media, strings.TrimSpace(r.FormValue("caption")), pos)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/gallery?flash=Foto+ditambah", http.StatusSeeOther)
}

func (a *App) cmsGalleryDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`DELETE FROM website_gallery WHERE id=? AND office_id=?`, id, a.officeOf(r))
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/gallery?flash=Dihapus", http.StatusSeeOther)
}

func (a *App) cmsTestimonials(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	rows, _ := a.db.Query(`SELECT id, name, content, status, verify_note, created_at FROM website_testimonials WHERE office_id=? ORDER BY id DESC LIMIT 100`, a.officeOf(r))
	type t struct {
		ID                      int64
		Name, Content, St, Ver, At string
	}
	var out []t
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x t
			_ = rows.Scan(&x.ID, &x.Name, &x.Content, &x.St, &x.Ver, &x.At)
			out = append(out, x)
		}
	}
	a.render(w, r, "cms_testimoni", "Testimoni", map[string]any{"Rows": out})
}

func (a *App) cmsTestimonialSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	_ = r.ParseForm()
	name, content := strings.TrimSpace(r.FormValue("name")), strings.TrimSpace(r.FormValue("content"))
	ver := strings.TrimSpace(r.FormValue("verify_note"))
	st := strings.ToUpper(strings.TrimSpace(r.FormValue("status")))
	if name == "" || content == "" {
		http.Error(w, "nama dan isi wajib diisi", http.StatusBadRequest)
		return
	}
	if st == "PUBLISHED" && ver == "" {
		http.Error(w, "terbitkan testimoni wajib mencantumkan bukti verifikasi (sumber ijin tampil)", http.StatusBadRequest)
		return
	}
	if st != "PUBLISHED" {
		st = "DRAFT"
	}
	if _, err := a.db.Exec(`INSERT INTO website_testimonials(office_id, name, content, status, verify_note) VALUES(?,?,?,?,?)`,
		a.officeOf(r), name, content, st, ver); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.Invalidate(a.officeOf(r))
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.testimonial.create", Entity: "testimonial", EntityID: "new", Detail: st})
	http.Redirect(w, r, "/cms/testimonials?flash=Testimoni+disimpan", http.StatusSeeOther)
}

func (a *App) cmsTestimonialStatus(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebEngagement) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	to := strings.ToUpper(strings.TrimSpace(r.FormValue("to")))
	if to != "PUBLISHED" && to != "DRAFT" {
		http.Error(w, "status tidak dikenal", http.StatusBadRequest)
		return
	}
	if to == "PUBLISHED" {
		var ver string
		_ = a.db.QueryRow(`SELECT verify_note FROM website_testimonials WHERE id=? AND office_id=?`, id, a.officeOf(r)).Scan(&ver)
		if strings.TrimSpace(ver) == "" {
			http.Error(w, "isi dulu bukti verifikasi sebelum menerbitkan", http.StatusBadRequest)
			return
		}
	}
	_, _ = a.db.Exec(`UPDATE website_testimonials SET status=? WHERE id=? AND office_id=?`, to, id, a.officeOf(r))
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/testimonials?flash=Status+diperbarui", http.StatusSeeOther)
}

// ---- versions ----

func (a *App) cmsVersions(w http.ResponseWriter, r *http.Request) {
	if !requireWebAny(w, r, a) {
		return
	}
	kind, ref := r.URL.Query().Get("kind"), r.URL.Query().Get("id")
	refID, _ := strconv.ParseInt(ref, 10, 64)
	q := `SELECT id, kind, ref_id, title, created_at FROM website_content_versions WHERE office_id=?`
	args := []any{a.officeOf(r)}
	if (kind == "page" || kind == "article") && refID > 0 {
		q += ` AND kind=? AND ref_id=?`
		args = append(args, kind, refID)
	}
	q += ` ORDER BY id DESC LIMIT 100`
	rows, _ := a.db.Query(q, args...)
	type v struct {
		ID             int64
		Kind           string
		Ref            int64
		Title, At      string
	}
	var out []v
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x v
			_ = rows.Scan(&x.ID, &x.Kind, &x.Ref, &x.Title, &x.At)
			out = append(out, x)
		}
	}
	a.render(w, r, "cms_versions", "Riwayat Versi Konten", map[string]any{"Rows": out, "Kind": kind, "Ref": ref})
}

func requireWebAny(w http.ResponseWriter, r *http.Request, a *App) bool {
	if !rbac.Has(mw.RoleOf(r), rbac.PermWebPagesView) && !rbac.Has(mw.RoleOf(r), rbac.PermWebArticles) {
		a.forbidden(w, r)
		return false
	}
	return true
}

func (a *App) cmsVersionRollback(w http.ResponseWriter, r *http.Request) {
	if !requireWebAny(w, r, a) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var kind string
	var refID int64
	var title, content string
	if err := a.db.QueryRow(`SELECT kind, ref_id, title, content FROM website_content_versions WHERE id=? AND office_id=?`,
		id, a.officeOf(r)).Scan(&kind, &refID, &title, &content); err != nil {
		http.NotFound(w, r)
		return
	}
	office := a.officeOf(r)
	if kind == "page" {
		if !rbac.Has(mw.RoleOf(r), rbac.PermWebPagesUpdate) {
			a.forbidden(w, r)
			return
		}
		var cur string
		_ = a.db.QueryRow(`SELECT content FROM website_pages WHERE id=? AND office_id=?`, refID, office).Scan(&cur)
		a.snapContentOffice(office, "page", refID, title+" (pra-rollback)", cur)
		if _, err := a.db.Exec(`UPDATE website_pages SET title=?, content=? WHERE id=? AND office_id=?`, title, content, refID, office); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		if !rbac.Has(mw.RoleOf(r), rbac.PermWebArticles) {
			a.forbidden(w, r)
			return
		}
		var cur string
		_ = a.db.QueryRow(`SELECT content FROM website_articles WHERE id=? AND office_id=?`, refID, office).Scan(&cur)
		a.snapContentOffice(office, "article", refID, title+" (pra-rollback)", cur)
		if _, err := a.db.Exec(`UPDATE website_articles SET title=?, content=? WHERE id=? AND office_id=?`, title, content, refID, office); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	a.web.Invalidate(office)
	a.web.RecordHistory(office, kind, refID, "rollback", "", "ROLLBACK", a.uid(r), "versi#"+strconv.FormatInt(id, 10), clientIPOf(r))
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.version.rollback", Entity: kind, EntityID: strconv.FormatInt(refID, 10), Detail: "versi#" + strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/cms/versions?kind="+kind+"&id="+strconv.FormatInt(refID, 10)+"&flash=Rollback+berhasil", http.StatusSeeOther)
}

// ---- quality dashboard ----

func (a *App) cmsQuality(w http.ResponseWriter, r *http.Request) {
	if !requireWebAny(w, r, a) {
		return
	}
	type row struct {
		Kind, Title, URL string
		Score            int
		Label            string
		Issues           []string
	}
	var out []row
	prows, _ := a.db.Query(`SELECT id, title, slug, content, seo_title, seo_desc FROM website_pages WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL`, a.officeOf(r))
	if prows != nil {
		defer prows.Close()
		for prows.Next() {
			var id int64
			var title, slug, content, st, sd string
			_ = prows.Scan(&id, &title, &slug, &content, &st, &sd)
			sc, lb := readabilityScore(content)
			var iss []string
			if st == "" {
				iss = append(iss, "tanpa SEO title")
			}
			if sd == "" {
				iss = append(iss, "tanpa SEO desc")
			}
			if len([]rune(stripTagsSimple(content))) < 300 {
				iss = append(iss, "konten < 300 karakter")
			}
			out = append(out, row{"Halaman", title, "/p/" + slug, sc, lb, iss})
			_ = id
		}
	}
	arows, _ := a.db.Query(`SELECT title, slug, content, seo_title, seo_desc FROM website_articles WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL`, a.officeOf(r))
	if arows != nil {
		defer arows.Close()
		for arows.Next() {
			var title, slug, content, st, sd string
			_ = arows.Scan(&title, &slug, &content, &st, &sd)
			sc, lb := readabilityScore(content)
			var iss []string
			if st == "" {
				iss = append(iss, "tanpa SEO title")
			}
			if sd == "" {
				iss = append(iss, "tanpa SEO desc")
			}
			if len([]rune(stripTagsSimple(content))) < 300 {
				iss = append(iss, "konten < 300 karakter")
			}
			out = append(out, row{"Artikel", title, "/artikel/" + slug, sc, lb, iss})
		}
	}
	a.render(w, r, "cms_quality", "Kualitas Konten", map[string]any{"Rows": out})
}

// ---- stats ----

func (a *App) cmsStats(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebAudit) {
		return
	}
	rows, _ := a.db.Query(`SELECT path, SUM(views) FROM website_pageviews WHERE office_id=? GROUP BY path ORDER BY 2 DESC LIMIT 50`, a.officeOf(r))
	type s struct {
		Path  string
		Views int
	}
	var out []s
	var total int
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x s
			_ = rows.Scan(&x.Path, &x.Views)
			total += x.Views
			out = append(out, x)
		}
	}
	drows, _ := a.db.Query(`SELECT day, SUM(views) FROM website_pageviews WHERE office_id=? AND day >= CURDATE() - INTERVAL 14 DAY GROUP BY day ORDER BY day`, a.officeOf(r))
	type d struct {
		Day   string
		Views int
	}
	var days []d
	if drows != nil {
		defer drows.Close()
		for drows.Next() {
			var x d
			_ = drows.Scan(&x.Day, &x.Views)
			days = append(days, x)
		}
	}
	a.render(w, r, "cms_stats", "Statistik Kunjungan", map[string]any{"Rows": out, "Days": days, "Total": total})
}

// ---- export ZIP (markdown) ----

func (a *App) cmsExport(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebsiteManage) {
		return
	}
	office := a.officeOf(r)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="konten-situs.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()
	write := func(name, body string) {
		f, err := zw.Create(name)
		if err != nil {
			return
		}
		_, _ = f.Write([]byte(body))
	}
	prows, _ := a.db.Query(`SELECT slug, title, content, status, locale FROM website_pages WHERE office_id=? AND deleted_at IS NULL`, office)
	if prows != nil {
		defer prows.Close()
		for prows.Next() {
			var slug, title, content, st, loc string
			_ = prows.Scan(&slug, &title, &content, &st, &loc)
			write(fmt.Sprintf("halaman/%s.%s.md", slug, loc), fmt.Sprintf("# %s\n\nStatus: %s\n\n%s\n", title, st, stripTagsSimple(content)))
		}
	}
	arows, _ := a.db.Query(`SELECT slug, title, content, status, locale FROM website_articles WHERE office_id=? AND deleted_at IS NULL`, office)
	if arows != nil {
		defer arows.Close()
		for arows.Next() {
			var slug, title, content, st, loc string
			_ = arows.Scan(&slug, &title, &content, &st, &loc)
			write(fmt.Sprintf("artikel/%s.%s.md", slug, loc), fmt.Sprintf("# %s\n\nStatus: %s\n\n%s\n", title, st, stripTagsSimple(content)))
		}
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.export", Entity: "site", EntityID: "all"})
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
