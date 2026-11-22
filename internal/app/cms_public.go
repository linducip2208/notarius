package app

// Public website renderer: reads ONLY published CMS rows (never admin tables).
// Office is resolved from the install (standalone: first office).

import (
	"database/sql"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/cms"
	mw "notaryos/internal/middleware"
)

func (a *App) pubOffice() int64 {
	var id int64
	if a.db != nil {
		_ = a.db.QueryRow(`SELECT COALESCE(MIN(id),0) FROM offices`).Scan(&id)
	}
	return id
}

func (a *App) pubBrand(office int64) BrandingView { return a.brandView(office) }

// pubBase is the common public data (brand, nav, site).
func (a *App) pubBase(office int64, r *http.Request) map[string]any {
	a.web.NoteHost(office, r.Host)
	nav, _ := a.web.PublicNav(office, "header")
	foot, _ := a.web.PublicNav(office, "footer")
	st, _ := a.web.SiteOf(office)
	var oname, ophone, oemail, oaddr, osite string
	_ = a.db.QueryRow(`SELECT name, phone, email, address, website FROM offices WHERE id=?`, office).Scan(&oname, &ophone, &oemail, &oaddr, &osite)
	lang := pubLang(r)
	var ann string
	var maint int
	_ = a.db.QueryRow(`SELECT announcement, maintenance FROM website_settings WHERE office_id=?`, office).Scan(&ann, &maint)
	return map[string]any{
		"Brand": a.pubBrand(office), "Nav": nav, "Foot": foot, "Site": st,
		"Office": map[string]any{"Name": oname, "Phone": ophone, "Email": oemail, "Addr": oaddr, "Site": osite},
		"Lang": lang, "OtherLang": map[string]any{"en": "id", "id": "en"}[lang], "IsEN": lang == "en",
		"Announcement": ann, "Maintenance": maint, "LoggedIn": mw.RoleOf(r) != "",
	}
}

func (a *App) publicHome(w http.ResponseWriter, r *http.Request) {
	if mw.RoleOf(r) != "" {
		http.Redirect(w, r, "/app", http.StatusSeeOther)
		return
	}
	office := a.pubOffice()
	data := a.pubBase(office, r)
	if office == 0 {
		if r.URL.Query().Get("cms") == "1" {
			data["HasCMS"] = false
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := a.tmpl.ExecuteTemplate(w, "pub_home_empty", map[string]any{"Title": "Beranda", "Data": data}); err != nil {
				http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
			}
			return
		}
		a.landing(w, r)
		return
	}
	if p, secs, err := a.web.PublicPage(office, "beranda", pubLang(r)); err == nil {
		data["Page"] = p
		data["Sections"] = secs
		data["CMSServices"], _ = a.web.PublicServices(office, pubLang(r))
		var nServices, nArticles, nFaq int
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_services WHERE office_id=? AND visible=1`, office).Scan(&nServices)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_articles WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL`, office).Scan(&nArticles)
		_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_faqs WHERE office_id=? AND visible=1`, office).Scan(&nFaq)
		data["Stats"] = map[string]int{"Services": nServices, "Articles": nArticles, "FAQ": nFaq}
		data["ServiceImages"] = []string{
			"/assets/landing/notarius-feat-akta.jpg",
			"/assets/landing/notarius-feat-dokumen.jpg",
			"/assets/landing/notarius-feat-klien.jpg",
			"/assets/landing/notarius-feat-keamanan.jpg",
		}
		data["HasCMS"] = true
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := a.tmpl.ExecuteTemplate(w, "pub_home", map[string]any{"Title": p.Title, "Data": data}); err != nil {
			a.log.Error("render", "page", "pub_home", "err", err)
			http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
		}
		return
	}
	// Fallback: legacy landing (logged-in users go to app).
	if r.URL.Query().Get("cms") == "1" {
		data["HasCMS"] = false
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := a.tmpl.ExecuteTemplate(w, "pub_home_empty", map[string]any{"Title": "Beranda", "Data": data}); err != nil {
			http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
		}
		return
	}
	a.landing(w, r)
}

func (a *App) publicPageSlug(slug string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		office := a.pubOffice()
		data := a.pubBase(office, r)
		if slug == "tentang-kami" {
			// Profil kantor selalu tampil (data publik); halaman CMS opsional memperkaya.
			if p, secs, err := a.web.PublicPage(office, slug, pubLang(r)); err == nil {
				data["Page"] = p
				data["Sections"] = secs
				data["HasPage"] = true
			}
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			if err := a.tmpl.ExecuteTemplate(w, "pub_about", map[string]any{"Title": "Tentang Kami", "Data": data}); err != nil {
				http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
			}
			return
		}
		p, secs, err := a.web.PublicPage(office, slug, pubLang(r))
		if err != nil {
			a.notFound(w, r)
			return
		}
		data["Page"] = p
		data["Sections"] = secs
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := a.tmpl.ExecuteTemplate(w, "pub_page", map[string]any{"Title": p.Title, "Data": data}); err != nil {
			http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
		}
	}
}

func (a *App) publicArticles(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page < 1 {
		page = 1
	}
	const per = 9
	list, total, _ := a.web.PublicArticles(office, pubLang(r), q.Get("q"), per, (page-1)*per)
	pages := (total + per - 1) / per
	data["List"] = list
	data["Total"] = total
	data["Pages"] = pages
	data["Page"] = page
	data["Q"] = q.Get("q")
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_articles", map[string]any{"Title": "Artikel", "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

func (a *App) publicArticle(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	art, rel, err := a.web.PublicArticle(office, chi.URLParam(r, "slug"), pubLang(r))
	if err != nil {
		a.notFound(w, r)
		return
	}
	data["Article"] = art
	data["Related"] = rel
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_article", map[string]any{"Title": art.Title, "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

func (a *App) publicFAQ(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	faqs, _ := a.web.PublicFAQs(office, pubLang(r))
	// group by category preserving order
	type group struct {
		Name  string
		Items []cms.FAQ
	}
	var groups []group
	idx := map[string]int{}
	for _, f := range faqs {
		if i, ok := idx[f.Category]; ok {
			groups[i].Items = append(groups[i].Items, f)
		} else {
			idx[f.Category] = len(groups)
			groups = append(groups, group{Name: f.Category, Items: []cms.FAQ{f}})
		}
	}
	data["Groups"] = groups
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_faq", map[string]any{"Title": "FAQ", "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

func (a *App) publicContact(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	if p, secs, err := a.web.PublicPage(office, "kontak", pubLang(r)); err == nil {
		data["Page"] = p
		data["Sections"] = secs
		data["HasPage"] = true
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_contact", map[string]any{"Title": "Kontak", "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// publicPrivacy renders the honest privacy notice (no tracking, local-only data).
func (a *App) publicPrivacy(w http.ResponseWriter, r *http.Request) {
	a.renderPub(w, r, "pub_privasi", "Kebijakan Privasi", nil)
}

func (a *App) publicServices(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	list, _ := a.web.PublicServices(office, pubLang(r))
	data["List"] = list
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_services", map[string]any{"Title": "Layanan Kami", "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

func (a *App) publicService(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	var sv cms.Service
	var img sql.NullInt64
	var code sql.NullString
	lang := pubLang(r)
	err := a.db.QueryRow(`SELECT id, name, slug, description, content, image_media_id, seo_title, seo_desc, service_code
		FROM website_services WHERE office_id=? AND slug=? AND locale=? AND visible=1`, office, chi.URLParam(r, "slug"), lang).Scan(
		&sv.ID, &sv.Name, &sv.Slug, &sv.Description, &sv.Content, &img, &sv.SEO.Title, &sv.SEO.Desc, &code)
	if err != nil {
		// fallback bahasa default
		err = a.db.QueryRow(`SELECT id, name, slug, description, content, image_media_id, seo_title, seo_desc, service_code
			FROM website_services WHERE office_id=? AND slug=? AND locale='id' AND visible=1`, office, chi.URLParam(r, "slug")).Scan(
			&sv.ID, &sv.Name, &sv.Slug, &sv.Description, &sv.Content, &img, &sv.SEO.Title, &sv.SEO.Desc, &code)
	}
	if err != nil {
		a.notFound(w, r)
		return
	}
	sv.ImageMediaID = img
	if code.Valid {
		sv.ServiceCode = code.String
	}
	data["Service"] = sv
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_service", map[string]any{"Title": sv.Name, "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// publicGeneric renders any other published page at /p/{slug}.
func (a *App) publicGeneric(w http.ResponseWriter, r *http.Request) {
	office := a.pubOffice()
	data := a.pubBase(office, r)
	p, secs, err := a.web.PublicPage(office, chi.URLParam(r, "slug"), pubLang(r))
	if err != nil {
		a.notFound(w, r)
		return
	}
	data["Page"] = p
	data["Sections"] = secs
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.tmpl.ExecuteTemplate(w, "pub_page", map[string]any{"Title": p.Title, "Data": data}); err != nil {
		http.Error(w, "gagal merender halaman", http.StatusInternalServerError)
	}
}

// cmsSitemapURLs returns extra public URLs from CMS (sitemap only lists these).
func (a *App) cmsSitemapURLs(base string) []string {
	office := a.pubOffice()
	if office == 0 || a.web == nil || a.db == nil {
		return nil
	}
	var out []string
	out = append(out, base+"/artikel", base+"/faq", base+"/layanan-kami", base+"/kontak", base+"/tentang-kami",
		base+"/kalkulator", base+"/konsultasi", base+"/booking", base+"/tim", base+"/galeri",
		base+"/testimoni", base+"/arsip", base+"/cari")
	pages, arts := a.web.SitemapURLs(office)
	static := map[string]bool{"beranda": true, "tentang-kami": true, "kontak": true, "artikel": true, "faq": true, "layanan-kami": true, "layanan": true, "docs": true}
	for _, sl := range pages {
		if !static[sl] {
			out = append(out, base+"/p/"+sl)
		}
	}
	for _, sl := range arts {
		out = append(out, base+"/artikel/"+sl)
	}
	rows, _ := a.db.Query(`SELECT slug FROM website_services WHERE office_id=? AND visible=1`, office)
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var sl string
			_ = rows.Scan(&sl)
			out = append(out, base+"/layanan-kami/"+sl)
		}
	}
	return out
}
