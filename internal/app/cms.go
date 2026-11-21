package app

// CMS admin: website dashboard, white-label branding, site/domain settings,
// media library, publication history, first-run setup. All office-scoped.

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/cms"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

func (a *App) requireWeb(w http.ResponseWriter, r *http.Request, perm string) bool {
	if !rbac.Has(mw.RoleOf(r), perm) {
		a.forbidden(w, r)
		return false
	}
	return true
}

// ---- website dashboard ----

func (a *App) cmsDashboard(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebsiteView) {
		return
	}
	office := a.officeOf(r)
	type counts struct {
		Pages, Published, Drafts, Articles, Media, Nav, FAQ, Services int
	}
	var c counts
	_ = a.db.QueryRow(`SELECT COUNT(*), SUM(status='PUBLISHED'), SUM(status IN ('DRAFT','REVIEW','SCHEDULED')) FROM website_pages WHERE office_id=? AND deleted_at IS NULL`, office).Scan(&c.Pages, &c.Published, &c.Drafts)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_articles WHERE office_id=? AND deleted_at IS NULL`, office).Scan(&c.Articles)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_media WHERE office_id=?`, office).Scan(&c.Media)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_nav_items WHERE office_id=?`, office).Scan(&c.Nav)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_faqs WHERE office_id=?`, office).Scan(&c.FAQ)
	_ = a.db.QueryRow(`SELECT COUNT(*) FROM website_services WHERE office_id=?`, office).Scan(&c.Services)
	// Missing SEO: published pages/articles without seo title+desc.
	type missing struct{ Kind, Title, ID int64 }
	_ = missing{}
	var miss []map[string]any
	prows, _ := a.db.Query(`SELECT id, title FROM website_pages WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL AND (seo_title='' OR seo_desc='') LIMIT 20`, office)
	if prows != nil {
		defer prows.Close()
		for prows.Next() {
			var id int64
			var t string
			_ = prows.Scan(&id, &t)
			miss = append(miss, map[string]any{"Kind": "Halaman", "ID": id, "Title": t})
		}
	}
	arows, _ := a.db.Query(`SELECT id, title FROM website_articles WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL AND (seo_title='' OR seo_desc='') LIMIT 20`, office)
	if arows != nil {
		defer arows.Close()
		for arows.Next() {
			var id int64
			var t string
			_ = arows.Scan(&id, &t)
			miss = append(miss, map[string]any{"Kind": "Artikel", "ID": id, "Title": t})
		}
	}
	hist, _ := a.web.History(office, 10)
	st, _ := a.web.SiteOf(office)
	a.render(w, r, "cms_dash", "Website", map[string]any{
		"Counts": c, "Missing": miss, "History": hist, "Site": st,
	})
}

// ---- branding ----

func (a *App) brandingPage(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebBranding) {
		return
	}
	b, _ := a.web.BrandingOf(a.officeOf(r))
	a.render(w, r, "cms_branding", "Branding", map[string]any{"B": b})
}

func (a *App) brandingSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebBranding) {
		return
	}
	_ = r.ParseForm()
	office := a.officeOf(r)
	radius, _ := strconv.Atoi(r.FormValue("border_radius"))
	b := cms.Branding{
		AppName: r.FormValue("app_name"), ShortName: r.FormValue("short_name"),
		Tagline: r.FormValue("tagline"), Primary: r.FormValue("primary"),
		Secondary: r.FormValue("secondary"), Accent: r.FormValue("accent"),
		SidebarBG: r.FormValue("sidebar_bg"), HeaderBG: r.FormValue("header_bg"),
		ButtonStyle: r.FormValue("button_style"), BorderRadius: radius,
		FontFamily: r.FormValue("font_family"), LogoRef: r.FormValue("logo_ref"),
		FaviconRef: r.FormValue("favicon_ref"), LoginBG: r.FormValue("login_bg"),
		FooterText: r.FormValue("footer_text"), CopyrightText: r.FormValue("copyright_text"),
		DefaultLang: r.FormValue("default_lang"), Timezone: r.FormValue("timezone"),
		DateFormat: r.FormValue("date_format"), Currency: r.FormValue("currency"),
	}
	if err := a.web.SaveBranding(office, b); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "branding.save", Entity: "office", EntityID: strconv.FormatInt(office, 10)})
	http.Redirect(w, r, "/cms/branding?flash=Branding+disimpan", http.StatusSeeOther)
}

// ---- site settings + domain ----

func (a *App) sitePage(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebSEO) {
		return
	}
	st, _ := a.web.SiteOf(a.officeOf(r))
	var name, phone, email, addr, site string
	_ = a.db.QueryRow(`SELECT name, phone, email, address, website FROM offices WHERE id=?`, a.officeOf(r)).Scan(&name, &phone, &email, &addr, &site)
	a.render(w, r, "cms_site", "Pengaturan Website", map[string]any{
		"Site": st, "Office": map[string]any{"Name": name, "Phone": phone, "Email": email, "Addr": addr, "Site": site},
	})
}

func (a *App) siteSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebSEO) {
		return
	}
	_ = r.ParseForm()
	office := a.officeOf(r)
	st, _ := a.web.SiteOf(office)
	st.SiteTitle, st.SiteDesc, st.DefaultOG = r.FormValue("site_title"), r.FormValue("site_desc"), r.FormValue("default_og")
	st.BaseURL = strings.TrimSpace(r.FormValue("base_url"))
	if err := a.web.SaveSite(office, st); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "website.seo.save", Entity: "office", EntityID: strconv.FormatInt(office, 10)})
	http.Redirect(w, r, "/cms/site?flash=Pengaturan+website+disimpan", http.StatusSeeOther)
}

func (a *App) domainSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebDomain) {
		return
	}
	_ = r.ParseForm()
	office := a.officeOf(r)
	st, _ := a.web.SiteOf(office)
	st.PrimaryDomain = strings.ToLower(strings.TrimSpace(r.FormValue("primary_domain")))
	st.EnforceHTTPS = r.FormValue("enforce_https") == "1"
	st.DomainVerified = r.FormValue("domain_verified") == "1"
	if err := a.web.SaveSite(office, st); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "website.domain.save", Entity: "office", EntityID: strconv.FormatInt(office, 10), Detail: st.PrimaryDomain})
	http.Redirect(w, r, "/cms/site?flash=Domain+disimpan", http.StatusSeeOther)
}

// ---- media library ----

func (a *App) mediaPage(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebMedia) {
		return
	}
	q := r.URL.Query()
	limit, offset := 24, 0
	if p, err := strconv.Atoi(q.Get("page")); err == nil && p > 1 {
		offset = (p - 1) * limit
	}
	list, total, _ := a.web.ListMedia(a.officeOf(r), q.Get("q"), q.Get("folder"), q.Get("mime"), limit, offset)
	pages := (total + limit - 1) / limit
	a.render(w, r, "cms_media", "Media Library", map[string]any{
		"Items": list, "Total": total, "Pages": pages, "Q": q.Get("q"),
		"Folder": q.Get("folder"), "MIME": q.Get("mime"),
	})
}

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func (a *App) mediaUpload(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebMedia) {
		return
	}
	if err := r.ParseMultipartForm(12 << 20); err != nil {
		http.Error(w, "berkas terlalu besar (maks 10 MB)", http.StatusBadRequest)
		return
	}
	f, hdr, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "pilih berkas", http.StatusBadRequest)
		return
	}
	defer f.Close()
	raw, err := io.ReadAll(io.LimitReader(f, (10<<20)+1))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if len(raw) == 0 || len(raw) > 10<<20 {
		http.Error(w, "ukuran tidak valid (maks 10 MB)", http.StatusBadRequest)
		return
	}
	head := raw
	if len(raw) > 512 {
		head = raw[:512]
	}
	mime, err := cms.ValidateMedia(hdr.Filename, int64(len(raw)), head)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Re-encode raster images (strip metadata/scripts); svg/webp/pdf pass through.
	safe, iw, ih, err := cms.Reencode(mime, raw)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.overQuota(a.officeOf(r), int64(len(safe))) {
		http.Error(w, "kuota penyimpanan kantor terlampaui — hubungi administrator", http.StatusInsufficientStorage)
		return
	}
	ref := cms.MediaRef(hdr.Filename, time.Now(), randHex(8))
	if err := a.files.Save(ref, bytes.NewReader(safe)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// Thumbnail (≤320px) for raster images, stored alongside (no DB row).
	if iw > 320 || ih > 320 {
		if th := downscale(safe, 320); th != nil {
			_ = a.files.Save(ref+".thumb", bytes.NewReader(th))
		}
	}
	office := a.officeOf(r)
	m := &cms.Media{Ref: ref, OriginalName: filepath.Base(hdr.Filename), MIME: mime,
		Size: int64(len(safe)), Width: iw, Height: ih,
		Alt: r.FormValue("alt"), Caption: r.FormValue("caption"), Folder: r.FormValue("folder")}
	id, err := a.web.SaveMedia(office, m, a.uid(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "media.upload", Entity: "media", EntityID: strconv.FormatInt(id, 10), Detail: hdr.Filename})
	http.Redirect(w, r, "/cms/media?flash=Media+diunggah", http.StatusSeeOther)
}

// downscale box-averages raw PNG/JPEG bytes to max width w (pure Go, no deps).
func downscale(raw []byte, w int) []byte {
	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil
	}
	b := img.Bounds()
	dw, dh := b.Dx(), b.Dy()
	if dw <= w {
		return nil
	}
	dh = dh * w / dw
	dw = w
	out := image.NewRGBA(image.Rect(0, 0, dw, dh))
	sx := float64(b.Dx()) / float64(dw)
	sy := float64(b.Dy()) / float64(dh)
	for y := 0; y < dh; y++ {
		for x := 0; x < dw; x++ {
			var r, g, bl, al, n uint32
			x0, x1 := int(float64(x)*sx), int(float64(x+1)*sx)
			y0, y1 := int(float64(y)*sy), int(float64(y+1)*sy)
			if x1 <= x0 {
				x1 = x0 + 1
			}
			if y1 <= y0 {
				y1 = y0 + 1
			}
			for yy := y0; yy < y1; yy++ {
				for xx := x0; xx < x1; xx++ {
					rr, gg, bb, aa := img.At(b.Min.X+xx, b.Min.Y+yy).RGBA()
					r, g, bl, al = r+rr, g+gg, bl+bb, al+aa
					n++
				}
			}
			out.Set(x, y, color.NRGBA64{R: uint16(r / n), G: uint16(g / n), B: uint16(bl / n), A: uint16(al / n)})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, out); err != nil {
		return nil
	}
	return buf.Bytes()
}

func (a *App) mediaDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebMedia) {
		return
	}
	_ = r.ParseForm()
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	ref, err := a.web.DeleteMedia(a.officeOf(r), id)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	// Best-effort byte removal (idempotent).
	_ = a.files.Delete(ref)
	_ = a.files.Delete(ref + ".thumb")
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "media.delete", Entity: "media", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/cms/media?flash=Media+dihapus", http.StatusSeeOther)
}

// mediaServe serves PUBLIC library bytes (no login). Only rows in website_media
// are reachable; private DMS files use a different store+routes entirely.
func (a *App) mediaServe(w http.ResponseWriter, r *http.Request) {
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var ref, mime string
	var office int64
	err := a.db.QueryRow(`SELECT ref, mime, office_id FROM website_media WHERE id=?`, id).Scan(&ref, &mime, &office)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if r.URL.Query().Get("thumb") == "1" {
		if rc, err := a.files.Open(ref + ".thumb"); err == nil {
			defer rc.Close()
			w.Header().Set("Content-Type", "image/png")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			_, _ = io.Copy(w, rc)
			return
		}
	}
	rc, err := a.files.Open(ref)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	if strings.HasPrefix(mime, "image/") {
		_, _ = io.Copy(w, rc)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+filepath.Base(ref)+`"`)
	_, _ = io.Copy(w, rc)
}

// mediaByRef serves branding assets (logo/favicon/login bg) by storage ref.
// The ref must belong to this install's website_media (no path traversal: exact match).
func (a *App) mediaByRef(w http.ResponseWriter, r *http.Request) {
	ref := r.URL.Query().Get("ref")
	if ref == "" || strings.Contains(ref, "..") {
		http.NotFound(w, r)
		return
	}
	var mime string
	var n int
	_ = a.db.QueryRow(`SELECT mime, COUNT(*) FROM website_media WHERE ref=?`, ref).Scan(&mime, &n)
	if n == 0 || mime == "" {
		http.NotFound(w, r)
		return
	}
	rc, err := a.files.Open(ref)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	defer rc.Close()
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = io.Copy(w, rc)
}

// ---- publication history ----

func (a *App) cmsHistory(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebAudit) {
		return
	}
	hist, _ := a.web.History(a.officeOf(r), 100)
	a.render(w, r, "cms_history", "Riwayat Publikasi", map[string]any{"Hist": hist})
}

// ---- first-run setup ----

func (a *App) setupPage(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebsiteManage) {
		return
	}
	st, _ := a.web.SiteOf(a.officeOf(r))
	b, _ := a.web.BrandingOf(a.officeOf(r))
	a.render(w, r, "cms_setup", "Setup Website", map[string]any{"Site": st, "B": b})
}

func (a *App) setupSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebsiteManage) {
		return
	}
	_ = r.ParseForm()
	office := a.officeOf(r)
	b, _ := a.web.BrandingOf(office)
	if v := r.FormValue("app_name"); v != "" {
		b.AppName = v
	}
	if v := r.FormValue("tagline"); v != "" {
		b.Tagline = v
	}
	b.Primary, b.Secondary, b.Accent = r.FormValue("primary"), r.FormValue("secondary"), r.FormValue("accent")
	if err := a.web.SaveBranding(office, b); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	st, _ := a.web.SiteOf(office)
	st.SiteTitle = r.FormValue("site_title")
	st.SiteDesc = r.FormValue("site_desc")
	st.BaseURL = strings.TrimSpace(r.FormValue("base_url"))
	st.SetupDone = true
	if err := a.web.SaveSite(office, st); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if _, err := a.db.Exec(`UPDATE offices SET phone=?, email=?, address=?, website=? WHERE id=?`,
		r.FormValue("phone"), r.FormValue("email"), r.FormValue("address"), r.FormValue("website"), office); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "website.setup", Entity: "office", EntityID: strconv.FormatInt(office, 10)})
	http.Redirect(w, r, "/cms?flash=Setup+selesai", http.StatusSeeOther)
}

// setupSample creates clearly-labeled CONTOH pages as DRAFT (never auto-published).
func (a *App) setupSample(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebsiteManage) {
		return
	}
	_ = r.ParseForm()
	office := a.officeOf(r)
	uid := a.uid(r)
	samples := []struct{ title, slug, excerpt string }{
		{"[CONTOH] Beranda", "beranda", "Ubah + terbitkan bila siap tampil publik."},
		{"[CONTOH] Tentang Kami", "tentang-kami", "Ganti dengan profil kantor sebenarnya."},
		{"[CONTOH] Kontak", "kontak", "Sesuaikan jam operasional + alamat."},
	}
	made := 0
	for _, s := range samples {
		id, err := a.web.SavePage(office, &cms.Page{Title: s.title, Slug: s.slug, Excerpt: s.excerpt,
			Content: "<p>" + s.excerpt + "</p>", Template: "page", Status: cms.StatusDraft}, uid)
		if err == nil && id > 0 {
			made++
			a.web.RecordHistory(office, "page", id, "sample.create", "", "DRAFT", uid, "konten contoh (draf)", clientIPOf(r))
		}
	}
	a.audit.Record(audit.Entry{ActorID: uid, Action: "website.setup.sample", Entity: "office", EntityID: strconv.FormatInt(office, 10), Detail: fmt.Sprintf("%d draf", made)})
	http.Redirect(w, r, "/cms/pages?flash=Konten+contoh+dibuat+sebagai+DRAF", http.StatusSeeOther)
}
