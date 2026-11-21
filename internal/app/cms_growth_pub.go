package app

// CMS growth (publik): kalkulator estimasi, konsultasi, booking, tim/galeri/testimoni,
// pencarian situs, arsip artikel, pratinjau draf bertoken, serta gerbang pemeliharaan.
// Semua angka dari DB (services + price_items); selalu rentang/estimasi, bukan harga final.

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/cryptoxx"
	mw "notaryos/internal/middleware"
	"notaryos/internal/money"
	"notaryos/internal/rbac"
	"notaryos/internal/validate"
)

// pubLang resolves 'en' only on explicit ?lang=en, else 'id'.
func pubLang(r *http.Request) string {
	if strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang"))) == "en" {
		return "en"
	}
	return "id"
}

// pubGate is middleware for public HTML routes: maintenance gate (503 for
// anonymous, banner for staff) + honest internal pageview counting.
func (a *App) pubGate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		office := a.pubOffice()
		var maint int
		var ann string
		if a.db != nil && office != 0 {
			_ = a.db.QueryRow(`SELECT maintenance, announcement FROM website_settings WHERE office_id=?`, office).Scan(&maint, &ann)
			if r.Method == http.MethodGet {
				_, _ = a.db.Exec(`INSERT INTO website_pageviews(office_id, path, day, views) VALUES(?,?,CURDATE(),1) ON DUPLICATE KEY UPDATE views=views+1`, office, r.URL.Path)
			}
		}
		_ = ann
		if maint == 1 && mw.RoleOf(r) == "" {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.Header().Set("Retry-After", "3600")
			w.WriteHeader(http.StatusServiceUnavailable)
			_ = a.tmpl.ExecuteTemplate(w, "pub_maintenance", map[string]any{"Title": "Pemeliharaan", "Data": map[string]any{}})
			return
		}
		next.ServeHTTP(w, r)
	})
}

// renderPub is a small helper for public templates (brand/nav/site already in pubBase).
func (a *App) renderPub(w http.ResponseWriter, r *http.Request, name, title string, extra map[string]any) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	for k, v := range extra {
		data[k] = v
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, name, map[string]any{"Title": title, "Data": data}); err != nil {
		a.log.Error("render", "page", name, "err", err)
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// ---- kalkulator estimasi transparan ----

// priceComponents returns active global price_items (PNBP/materai/etc).
func (a *App) priceComponents() ([]map[string]any, error) {
	rows, err := a.db.Query(`SELECT name, amount FROM price_items WHERE is_active=1 ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var name, amount string
		_ = rows.Scan(&name, &amount)
		var n int64
		if v, err := strconv.ParseInt(strings.Split(amount, ".")[0], 10, 64); err == nil {
			n = v
		}
		out = append(out, map[string]any{"Name": name, "Amount": n, "Fmt": money.FormatIDR(n)})
	}
	return out, rows.Err()
}

func (a *App) pubKalkulator(w http.ResponseWriter, r *http.Request) {
	code := strings.TrimSpace(r.URL.Query().Get("code"))
	type opt struct{ Code, Name string }
	var opts []opt
	rows, _ := a.db.Query(`SELECT code, name FROM services WHERE is_active=1 ORDER BY name`)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var o opt
			_ = rows.Scan(&o.Code, &o.Name)
			opts = append(opts, o)
		}
	}
	data := map[string]any{"Opts": opts, "Code": code}
	if code != "" {
		var name, price, reqj string
		if err := a.db.QueryRow(`SELECT name, base_price, COALESCE(requirements,'') FROM services WHERE code=? AND is_active=1`, code).Scan(&name, &price, &reqj); err == nil {
			var base int64
			if n, err := strconv.ParseInt(strings.Split(price, ".")[0], 10, 64); err == nil {
				base = n
			}
			comps, _ := a.priceComponents()
			var add int64
			for _, c := range comps {
				if v, ok := c["Amount"].(int64); ok {
					add += v
				}
			}
			data["Name"] = name
			data["Base"] = base
			data["BaseFmt"] = money.FormatIDR(base)
			data["Comps"] = comps
			data["Add"] = add
			data["AddFmt"] = money.FormatIDR(add)
			data["Total"] = base + add
			data["TotalFmt"] = money.FormatIDR(base + add)
		}
	}
	a.renderPub(w, r, "pub_kalkulator", "Estimasi Biaya", data)
}

// ---- konsultasi masuk ----

func (a *App) pubKonsultasiForm(w http.ResponseWriter, r *http.Request) {
	a.renderPub(w, r, "pub_konsultasi", "Konsultasi", map[string]any{"Sent": r.URL.Query().Get("sent") == "1"})
}

func (a *App) pubKonsultasiSave(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	name, contact := strings.TrimSpace(r.FormValue("name")), strings.TrimSpace(r.FormValue("contact"))
	msg := strings.TrimSpace(r.FormValue("message"))
	svc := strings.TrimSpace(r.FormValue("service"))
	if name == "" || contact == "" || msg == "" {
		http.Error(w, "nama, kontak, dan kebutuhan wajib diisi", http.StatusBadRequest)
		return
	}
	if len(msg) > 2000 {
		http.Error(w, "pesan maksimal 2000 karakter", http.StatusBadRequest)
		return
	}
	office := a.pubOffice()
	if office == 0 {
		http.Error(w, "kantor belum dikonfigurasi", http.StatusServiceUnavailable)
		return
	}
	if _, err := a.db.Exec(`INSERT INTO website_inquiries(office_id, name, contact, service, message) VALUES(?,?,?,?,?)`,
		office, name, contact, svc, msg); err != nil {
		http.Error(w, "gagal menyimpan, coba lagi", http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/konsultasi?sent=1", http.StatusSeeOther)
}

// ---- booking jadwal ----

func (a *App) pubBookingForm(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	type slot struct {
		ID            int64
		Weekday, Span string
		Quota         int
	}
	var slots []slot
	dayName := map[int]string{0: "Minggu", 1: "Senin", 2: "Selasa", 3: "Rabu", 4: "Kamis", 5: "Jumat", 6: "Sabtu"}
	rows, _ := a.db.Query(`SELECT id, weekday, DATE_FORMAT(start_time,'%H:%i'), DATE_FORMAT(end_time,'%H:%i'), quota FROM website_slots WHERE office_id=? AND is_active=1 ORDER BY weekday, start_time`, office)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var s slot
			var wd int
			var st, en string
			_ = rows.Scan(&s.ID, &wd, &st, &en, &s.Quota)
			s.Weekday = dayName[wd%7]
			s.Span = st + "â€“" + en
			slots = append(slots, s)
		}
	}
	a.renderPub(w, r, "pub_booking", "Booking Jadwal", map[string]any{
		"Slots": slots, "Sent": r.URL.Query().Get("sent") == "1", "Err": r.URL.Query().Get("err"),
		"Min": time.Now().Format("2006-01-02"), "Max": time.Now().AddDate(0, 2, 0).Format("2006-01-02"),
	})
}

func (a *App) pubBookingSave(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	office := a.pubOffice()
	name, contact := strings.TrimSpace(r.FormValue("name")), strings.TrimSpace(r.FormValue("contact"))
	date, start := strings.TrimSpace(r.FormValue("date")), strings.TrimSpace(r.FormValue("start"))
	purpose := strings.TrimSpace(r.FormValue("purpose"))
	fail := func(msg string) {
		http.Redirect(w, r, "/booking?err="+msg, http.StatusSeeOther)
	}
	if name == "" || contact == "" || date == "" || start == "" {
		fail("nama,+kontak,+tanggal,+jam+wajib+diisi")
		return
	}
	// Booking bermuara ke janji temu + klien tamu: kontak harus nomor HP valid.
	// Selain HP (mis. email saja), gunakan formulir konsultasi.
	if validate.Phone(contact) != nil {
		fail("kontak+harus+nomor+HP+valid+(cth.+0812xxxxxxx)")
		return
	}
	day, err := time.Parse("2006-01-02", date)
	if err != nil || day.Before(time.Now().Truncate(24*time.Hour)) {
		fail("tanggal+tidak+valid")
		return
	}
	if day.After(time.Now().AddDate(0, 2, 0)) {
		fail("maksimal+2+bulan+ke+depan")
		return
	}
	// Kuota per slot harian: hitung booking aktif pada tanggal+jam yang sama.
	var used int
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_bookings WHERE office_id=? AND book_date=? AND start_time=? AND status IN ('MENUNGGU','DISETUJUI')`,
		office, date, start).Scan(&used)
	var quota int
	_ = a.db.QueryRow(`SELECT COALESCE(MAX(quota),4) FROM website_slots WHERE office_id=? AND is_active=1 AND weekday=DAYOFWEEK(?)-1`, office, date).Scan(&quota)
	if quota < 1 {
		quota = 4
	}
	if used >= quota {
		fail("slot+penuh,+pilih+jam+lain")
		return
	}
	if _, err := a.db.Exec(`INSERT INTO website_bookings(office_id, book_date, start_time, name, contact, purpose) VALUES(?,?,?,?,?,?)`,
		office, date, start, name, contact, purpose); err != nil {
		fail("gagal+menyimpan")
		return
	}
	http.Redirect(w, r, "/booking?sent=1", http.StatusSeeOther)
}

// ---- tim, galeri, testimoni ----

func (a *App) pubTeam(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	type m struct {
		Name, Title string
		Photo       int64
		HasPhoto    bool
	}
	var out []m
	rows, _ := a.db.Query(`SELECT name, title, photo_media_id FROM website_team WHERE office_id=? AND visible=1 ORDER BY position, id`, office)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x m
			var ph sql.NullInt64
			_ = rows.Scan(&x.Name, &x.Title, &ph)
			if ph.Valid {
				x.Photo, x.HasPhoto = ph.Int64, true
			}
			out = append(out, x)
		}
	}
	a.renderPub(w, r, "pub_team", "Tim Kami", map[string]any{"Rows": out})
}

func (a *App) pubGallery(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	type g struct {
		Media   int64
		Caption string
	}
	var out []g
	rows, _ := a.db.Query(`SELECT media_id, caption FROM website_gallery WHERE office_id=? ORDER BY position, id`, office)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x g
			_ = rows.Scan(&x.Media, &x.Caption)
			out = append(out, x)
		}
	}
	a.renderPub(w, r, "pub_gallery", "Galeri", map[string]any{"Rows": out})
}

func (a *App) pubTestimonials(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	type t struct {
		Name, Content, Verify string
	}
	var out []t
	rows, _ := a.db.Query(`SELECT name, content, verify_note FROM website_testimonials WHERE office_id=? AND status='PUBLISHED' ORDER BY id DESC LIMIT 30`, office)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x t
			_ = rows.Scan(&x.Name, &x.Content, &x.Verify)
			out = append(out, x)
		}
	}
	a.renderPub(w, r, "pub_testimoni", "Kata Klien", map[string]any{"Rows": out})
}

// ---- pencarian situs ----

func (a *App) pubSearch(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	lang := pubLang(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	type hit struct {
		Kind, Title, URL, Snip string
	}
	var out []hit
	if q != "" && len(q) <= 100 {
		like := "%" + q + "%"
		arows, _ := a.db.Query(`SELECT title, slug, excerpt FROM website_articles WHERE office_id=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL AND (title LIKE ? OR excerpt LIKE ? OR content LIKE ?) LIMIT 10`, office, lang, like, like, like)
		if arows != nil {
			func() {
				defer arows.Close()
				for arows.Next() {
					var h hit
					var slug string
					_ = arows.Scan(&h.Title, &slug, &h.Snip)
					h.Kind, h.URL = "Artikel", "/artikel/"+slug
					out = append(out, h)
				}
			}()
		}
		prows, _ := a.db.Query(`SELECT title, slug, excerpt FROM website_pages WHERE office_id=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL AND (title LIKE ? OR excerpt LIKE ? OR content LIKE ?) LIMIT 10`, office, lang, like, like, like)
		if prows != nil {
			func() {
				defer prows.Close()
				for prows.Next() {
					var h hit
					var slug string
					_ = prows.Scan(&h.Title, &slug, &h.Snip)
					h.Kind, h.URL = "Halaman", "/p/"+slug
					out = append(out, h)
				}
			}()
		}
		srows, _ := a.db.Query(`SELECT name, slug, description FROM website_services WHERE office_id=? AND locale=? AND visible=1 AND (name LIKE ? OR description LIKE ? OR content LIKE ?) LIMIT 10`, office, lang, like, like, like)
		if srows != nil {
			func() {
				defer srows.Close()
				for srows.Next() {
					var h hit
					var slug string
					_ = srows.Scan(&h.Title, &slug, &h.Snip)
					h.Kind, h.URL = "Layanan", "/layanan-kami/"+slug
					out = append(out, h)
				}
			}()
		}
		frows, _ := a.db.Query(`SELECT question, answer FROM website_faqs WHERE office_id=? AND locale=? AND visible=1 AND (question LIKE ? OR answer LIKE ?) LIMIT 10`, office, lang, like, like)
		if frows != nil {
			func() {
				defer frows.Close()
				for frows.Next() {
					var h hit
					_ = frows.Scan(&h.Title, &h.Snip)
					if len(h.Snip) > 140 {
						h.Snip = h.Snip[:140] + "â€¦"
					}
					h.Kind, h.URL = "FAQ", "/faq"
					out = append(out, h)
				}
			}()
		}
	}
	a.renderPub(w, r, "pub_cari", "Pencarian", map[string]any{"Q": q, "Hits": out})
}

// ---- arsip artikel ----

func (a *App) pubArchive(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	lang := pubLang(r)
	month := strings.TrimSpace(chi.URLParam(r, "month")) // YYYY-MM
	type a2 struct {
		Title, Slug, Date string
	}
	var out []a2
	var title string
	if month != "" {
		rows, _ := a.db.Query(`SELECT title, slug, DATE_FORMAT(published_at,'%d %b %Y') FROM website_articles WHERE office_id=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL AND DATE_FORMAT(published_at,'%Y-%m')=? ORDER BY published_at DESC`, office, lang, month)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var x a2
				_ = rows.Scan(&x.Title, &x.Slug, &x.Date)
				out = append(out, x)
			}
		}
		title = "Arsip " + month
	} else {
		type m struct {
			Month, Label string
			Count        int
		}
		var months []m
		rows, _ := a.db.Query(`SELECT DATE_FORMAT(published_at,'%Y-%m'), DATE_FORMAT(published_at,'%b %Y'), COUNT(*) FROM website_articles WHERE office_id=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL GROUP BY 1,2 ORDER BY 1 DESC`, office, lang)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var x m
				_ = rows.Scan(&x.Month, &x.Label, &x.Count)
				months = append(months, x)
			}
		}
		a.renderPub(w, r, "pub_arsip", "Arsip Artikel", map[string]any{"Months": months})
		return
	}
	a.renderPub(w, r, "pub_arsip_list", title, map[string]any{"Month": month, "Rows": out})
}

// ---- pratinjau draf bertoken (staf saja, HMAC) ----

func (a *App) previewToken(kind string, id int64) string {
	key, err := cryptoxx.Key()
	if err != nil {
		return ""
	}
	mac := hmac.New(sha256.New, key)
	fmt.Fprintf(mac, "preview:%s:%d", kind, id)
	return fmt.Sprintf("%x", mac.Sum(nil))
}

func (a *App) pubPreview(w http.ResponseWriter, r *http.Request) {
	if !rbac.Has(mw.RoleOf(r), rbac.PermWebPagesView) {
		a.forbidden(w, r)
		return
	}
	kind, idStr := r.URL.Query().Get("kind"), r.URL.Query().Get("id")
	id, _ := strconv.ParseInt(idStr, 10, 64)
	if (kind != "page" && kind != "article") || id <= 0 || r.URL.Query().Get("token") != a.previewToken(kind, id) {
		http.Error(w, "tautan pratinjau tidak valid", http.StatusForbidden)
		return
	}
	office := a.officeOf(r)
	if kind == "page" {
		p, secs, err := a.loadPage(office, id)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		a.renderPub(w, r, "pub_preview_page", "Pratinjau: "+p.Title, map[string]any{"Page": p, "Sections": secs, "Draft": true})
		return
	}
	var art struct {
		Title, Excerpt, Content string
	}
	err := a.db.QueryRow(`SELECT title, excerpt, content FROM website_articles WHERE id=? AND office_id=? AND deleted_at IS NULL`, id, office).Scan(&art.Title, &art.Excerpt, &art.Content)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.renderPub(w, r, "pub_preview_article", "Pratinjau: "+art.Title, map[string]any{"Article": art, "Draft": true})
}

// ---- keterbacaan (skor Flesch-ish sederhana, tanpa klaim palsu) ----

// readabilityScore returns 0-100 (higher = easier) + label, computed from
// average sentence length and syllable-ish heuristic on latin text.
func readabilityScore(text string) (int, string) {
	plain := stripTagsSimple(text)
	words := strings.Fields(plain)
	if len(words) == 0 {
		return 0, "kosong"
	}
	sents := 1
	for _, ch := range plain {
		if ch == '.' || ch == '!' || ch == '?' || ch == '\n' {
			sents++
		}
	}
	avgLen := float64(len(words)) / float64(sents)
	score := int(100 - (avgLen-8)*6)
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}
	label := "sulit"
	switch {
	case score >= 70:
		label = "mudah"
	case score >= 50:
		label = "sedang"
	}
	return score, label
}

func stripTagsSimple(s string) string {
	var b strings.Builder
	inTag := false
	for _, r := range s {
		switch {
		case r == '<':
			inTag = true
		case r == '>':
			inTag = false
		case !inTag:
			b.WriteRune(r)
		}
	}
	return b.String()
}

// snapContentOffice stores a version snapshot before update (kept: last 20 per doc).
func (a *App) snapContentOffice(officeID int64, kind string, refID int64, title, content string) {
	if a.db == nil || (kind != "page" && kind != "article") {
		return
	}
	_, _ = a.db.Exec(`INSERT INTO website_content_versions(office_id, kind, ref_id, title, content) VALUES(?,?,?,?,?)`,
		officeID, kind, refID, title, content)
	_, _ = a.db.Exec(`DELETE FROM website_content_versions WHERE office_id=? AND kind=? AND ref_id=? AND id NOT IN (
		SELECT id FROM (SELECT id FROM website_content_versions WHERE office_id=? AND kind=? AND ref_id=? ORDER BY id DESC LIMIT 20) t)`,
		officeID, kind, refID, officeID, kind, refID)
}
