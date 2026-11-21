package app

// CMS content admin: pages (+sections, duplicate, soft-delete/restore),
// articles, categories, services, FAQ, navigation. Status workflow enforced.

import (
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"notaryos/internal/audit"
	"notaryos/internal/cms"
	mw "notaryos/internal/middleware"
	"notaryos/internal/rbac"
)

func (a *App) uidPtr(r *http.Request) *int64 { return a.uid(r) }

// parseWebTime parses HTML datetime-local input.
func parseWebTime(v string) (t time.Time, err error) {
	for _, layout := range []string{"2006-01-02T15:04", "2006-01-02 15:04", "2006-01-02"} {
		if t, err = time.ParseInLocation(layout, v, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("waktu tidak valid")
}

// ---- pages ----

func (a *App) cmsPages(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesView) {
		return
	}
	q := r.URL.Query()
	rows, _ := a.db.Query(`SELECT id, title, slug, status, updated_at, published_at FROM website_pages
		WHERE office_id=? AND deleted_at IS NULL AND (title LIKE ? OR slug LIKE ?)
		ORDER BY updated_at DESC LIMIT 100`, a.officeOf(r), "%"+q.Get("q")+"%", "%"+q.Get("q")+"%")
	type row struct {
		ID, Title, Slug, Status, Updated, Published string
	}
	var out []row
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x row
			var pub sql.NullString
			_ = rows.Scan(&x.ID, &x.Title, &x.Slug, &x.Status, &x.Updated, &pub)
			if pub.Valid {
				x.Published = pub.String
			}
			out = append(out, x)
		}
	}
	a.render(w, r, "cms_pages", "Halaman Website", map[string]any{"Rows": out, "Q": q.Get("q")})
}

func (a *App) cmsPageNew(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesCreate) {
		return
	}
	a.render(w, r, "cms_page_form", "Halaman Baru", map[string]any{"IsNew": true})
}

func (a *App) pageFromForm(r *http.Request, id int64) *cms.Page {
	_ = r.ParseForm()
	var feat sql.NullInt64
	if v := r.FormValue("featured_media_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			feat = sql.NullInt64{Int64: n, Valid: true}
		}
	}
	var sched sql.NullTime
	if v := strings.TrimSpace(r.FormValue("scheduled_at")); v != "" {
		if t, err := parseWebTime(v); err == nil {
			sched = sql.NullTime{Time: t, Valid: true}
		}
	}
	return &cms.Page{
		ID: id, Title: strings.TrimSpace(r.FormValue("title")), Slug: strings.TrimSpace(r.FormValue("slug")),
		Excerpt: r.FormValue("excerpt"), Content: cms.SanitizeHTML(r.FormValue("content")),
		FeaturedMediaID: feat, Template: "page", Status: strings.TrimSpace(r.FormValue("status")),
		SEO: cms.SEO{Title: r.FormValue("seo_title"), Desc: r.FormValue("seo_desc"),
			Canonical: strings.TrimSpace(r.FormValue("canonical_url")), OGImage: r.FormValue("og_image"),
			Robots: r.FormValue("robots")},
		ScheduledAt: sched,
	}
}

func (a *App) cmsPageCreate(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesCreate) {
		return
	}
	p := a.pageFromForm(r, 0)
	if p.Status == "" {
		p.Status = cms.StatusDraft
	}
	if p.Status == cms.StatusPublished && !rbac.Has(mw.RoleOf(r), rbac.PermWebPagesPublish) {
		p.Status = cms.StatusReview
	}
	id, err := a.web.SavePage(a.officeOf(r), p, a.uidPtr(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.RecordHistory(a.officeOf(r), "page", id, "create", "", p.Status, a.uid(r), "", clientIPOf(r))
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.page.create", Entity: "page", EntityID: strconv.FormatInt(id, 10)})
	http.Redirect(w, r, "/cms/pages/"+strconv.FormatInt(id, 10)+"?flash=Halaman+dibuat", http.StatusSeeOther)
}

func (a *App) cmsPageEdit(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesView) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	p, secs, err := a.loadPage(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	a.render(w, r, "cms_page_form", "Edit Halaman", map[string]any{"P": p, "Secs": secs})
}

func (a *App) loadPage(officeID, id int64) (cms.Page, []cms.Section, error) {
	var p cms.Page
	var feat sql.NullInt64
	var sched, pub sql.NullTime
	var created, updated string
	err := a.db.QueryRow(`SELECT id, title, slug, excerpt, content, featured_media_id, template,
		status, seo_title, seo_desc, canonical_url, og_image, robots, created_at, updated_at,
		scheduled_at, published_at FROM website_pages WHERE id=? AND office_id=? AND deleted_at IS NULL`,
		id, officeID).Scan(&p.ID, &p.Title, &p.Slug, &p.Excerpt, &p.Content, &feat, &p.Template,
		&p.Status, &p.SEO.Title, &p.SEO.Desc, &p.SEO.Canonical, &p.SEO.OGImage, &p.SEO.Robots,
		&created, &updated, &sched, &pub)
	if err != nil {
		return p, nil, err
	}
	p.FeaturedMediaID, p.ScheduledAt, p.PublishedAt = feat, sched, pub
	secs, _ := a.web.SectionsOf(id, false)
	return p, secs, nil
}

func (a *App) cmsPageUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesUpdate) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	old, _, err := a.loadPage(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	p := a.pageFromForm(r, id)
	if p.Status == "" {
		p.Status = old.Status
	}
	if p.Status == cms.StatusPublished && old.Status != cms.StatusPublished &&
		!rbac.Has(mw.RoleOf(r), rbac.PermWebPagesPublish) {
		http.Error(w, "butuh izin publikasi", http.StatusForbidden)
		return
	}
	if _, err := a.web.SavePage(a.officeOf(r), p, a.uidPtr(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.RecordHistory(a.officeOf(r), "page", id, "update", old.Status, p.Status, a.uid(r), "", clientIPOf(r))
	a.audit.Record(audit.Entry{ActorID: a.uid(r), Action: "cms.page.update", Entity: "page", EntityID: strconv.FormatInt(id, 10), Detail: old.Status + "→" + p.Status})
	http.Redirect(w, r, "/cms/pages/"+strconv.FormatInt(id, 10)+"?flash=Halaman+disimpan", http.StatusSeeOther)
}

func (a *App) cmsPageDuplicate(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesCreate) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	old, secs, err := a.loadPage(a.officeOf(r), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	office := a.officeOf(r)
	old.ID = 0
	old.Slug = old.Slug + "-salinan"
	old.Title = old.Title + " (salinan)"
	old.Status = cms.StatusDraft
	nid, err := a.web.SavePage(office, &old, a.uidPtr(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	for _, s := range secs {
		s.ID = 0
		s.PageID = nid
		_, _ = a.web.SaveSection(office, &s)
	}
	a.web.RecordHistory(office, "page", nid, "duplicate", "", "DRAFT", a.uid(r), fmt.Sprintf("dari #%d", id), clientIPOf(r))
	http.Redirect(w, r, "/cms/pages/"+strconv.FormatInt(nid, 10)+"?flash=Duplikat+dibuat", http.StatusSeeOther)
}

func (a *App) cmsPageDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesDelete) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if _, err := a.db.Exec(`UPDATE website_pages SET deleted_at=UTC_TIMESTAMP() WHERE id=? AND office_id=?`, id, a.officeOf(r)); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	a.web.RecordHistory(a.officeOf(r), "page", id, "archive", "", "ARCHIVED", a.uid(r), "soft delete", clientIPOf(r))
	http.Redirect(w, r, "/cms/pages?flash=Halaman+diarsipkan", http.StatusSeeOther)
}

// ---- sections ----

func (a *App) cmsSectionSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesUpdate) {
		return
	}
	_ = r.ParseForm()
	pid, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	pos, _ := strconv.Atoi(r.FormValue("position"))
	vis := r.FormValue("visible") == "1" || r.FormValue("visible") == "on"
	sec := &cms.Section{PageID: pid, Type: r.FormValue("type"), Title: r.FormValue("title"),
		Subtitle: r.FormValue("subtitle"), Content: cms.SanitizeHTML(r.FormValue("content")),
		CTALabel: r.FormValue("cta_label"), CTAURL: strings.TrimSpace(r.FormValue("cta_url")),
		Background: r.FormValue("background"), Align: r.FormValue("align"), Layout: r.FormValue("layout"),
		Visible: vis, Position: pos}
	if v := r.FormValue("image_media_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			sec.ImageMediaID = sql.NullInt64{Int64: n, Valid: true}
		}
	}
	if _, err := a.web.SaveSection(a.officeOf(r), sec); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/cms/pages/"+strconv.FormatInt(pid, 10)+"?flash=Section+disimpan", http.StatusSeeOther)
}

func (a *App) cmsSectionDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebPagesUpdate) {
		return
	}
	sid, _ := strconv.ParseInt(chi.URLParam(r, "sid"), 10, 64)
	var pid int64
	_ = a.db.QueryRow(`SELECT page_id FROM website_page_sections WHERE id=? AND office_id=?`, sid, a.officeOf(r)).Scan(&pid)
	if pid == 0 {
		http.NotFound(w, r)
		return
	}
	_, _ = a.db.Exec(`DELETE FROM website_page_sections WHERE id=?`, sid)
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/pages/"+strconv.FormatInt(pid, 10)+"?flash=Section+dihapus", http.StatusSeeOther)
}

// ---- articles ----

func (a *App) cmsArticles(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	q := r.URL.Query()
	rows, _ := a.db.Query(`SELECT a.id, a.title, a.slug, a.status, COALESCE(c.name,''), a.updated_at
		FROM website_articles a LEFT JOIN website_categories c ON c.id=a.category_id
		WHERE a.office_id=? AND a.deleted_at IS NULL AND (a.title LIKE ? OR a.slug LIKE ?)
		ORDER BY a.updated_at DESC LIMIT 100`, a.officeOf(r), "%"+q.Get("q")+"%", "%"+q.Get("q")+"%")
	type row struct {
		ID                    int64
		Title, Slug, Status    string
		Cat, Updated           string
	}
	var out []row
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var x row
			_ = rows.Scan(&x.ID, &x.Title, &x.Slug, &x.Status, &x.Cat, &x.Updated)
			out = append(out, x)
		}
	}
	cats, _ := a.db.Query(`SELECT id, name FROM website_categories WHERE office_id=? ORDER BY name`, a.officeOf(r))
	var cl []map[string]any
	if cats != nil {
		defer cats.Close()
		for cats.Next() {
			var id int64
			var n string
			_ = cats.Scan(&id, &n)
			cl = append(cl, map[string]any{"ID": id, "Name": n})
		}
	}
	a.render(w, r, "cms_articles", "Artikel", map[string]any{"Rows": out, "Q": q.Get("q"), "Cats": cl})
}

func (a *App) cmsArticleNew(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	cats, _ := a.db.Query(`SELECT id, name FROM website_categories WHERE office_id=? ORDER BY name`, a.officeOf(r))
	var cl []map[string]any
	if cats != nil {
		defer cats.Close()
		for cats.Next() {
			var id int64
			var n string
			_ = cats.Scan(&id, &n)
			cl = append(cl, map[string]any{"ID": id, "Name": n})
		}
	}
	a.render(w, r, "cms_article_form", "Artikel Baru", map[string]any{"IsNew": true, "Cats": cl})
}

func (a *App) cmsArticleEdit(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var art cms.Article
	var cat, cover sql.NullInt64
	var sched, pub sql.NullTime
	err := a.db.QueryRow(`SELECT id, category_id, title, slug, excerpt, content, cover_media_id,
		tags, author, status, seo_title, seo_desc, canonical_url, og_image, robots,
		scheduled_at, published_at FROM website_articles WHERE id=? AND office_id=? AND deleted_at IS NULL`,
		id, a.officeOf(r)).Scan(&art.ID, &cat, &art.Title, &art.Slug, &art.Excerpt, &art.Content,
		&cover, &art.Tags, &art.Author, &art.Status, &art.SEO.Title, &art.SEO.Desc,
		&art.SEO.Canonical, &art.SEO.OGImage, &art.SEO.Robots, &sched, &pub)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	art.CategoryID, art.CoverMediaID, art.ScheduledAt, art.PublishedAt = cat, cover, sched, pub
	cats, _ := a.db.Query(`SELECT id, name FROM website_categories WHERE office_id=? ORDER BY name`, a.officeOf(r))
	var cl []map[string]any
	if cats != nil {
		defer cats.Close()
		for cats.Next() {
			var cid int64
			var n string
			_ = cats.Scan(&cid, &n)
			cl = append(cl, map[string]any{"ID": cid, "Name": n})
		}
	}
	a.render(w, r, "cms_article_form", "Edit Artikel", map[string]any{"A": art, "Cats": cl})
}

func articleFromForm(r *http.Request, id int64) *cms.Article {
	_ = r.ParseForm()
	var cat, cover sql.NullInt64
	if v := r.FormValue("category_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cat = sql.NullInt64{Int64: n, Valid: true}
		}
	}
	if v := r.FormValue("cover_media_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			cover = sql.NullInt64{Int64: n, Valid: true}
		}
	}
	var sched sql.NullTime
	if v := strings.TrimSpace(r.FormValue("scheduled_at")); v != "" {
		if t, err := parseWebTime(v); err == nil {
			sched = sql.NullTime{Time: t, Valid: true}
		}
	}
	return &cms.Article{ID: id, CategoryID: cat, Title: strings.TrimSpace(r.FormValue("title")),
		Slug: strings.TrimSpace(r.FormValue("slug")), Excerpt: r.FormValue("excerpt"),
		Content: cms.SanitizeHTML(r.FormValue("content")), CoverMediaID: cover,
		Tags: r.FormValue("tags"), Author: r.FormValue("author"), Status: strings.TrimSpace(r.FormValue("status")),
		SEO: cms.SEO{Title: r.FormValue("seo_title"), Desc: r.FormValue("seo_desc"),
			Canonical: strings.TrimSpace(r.FormValue("canonical_url")), OGImage: r.FormValue("og_image"),
			Robots: r.FormValue("robots")},
		ScheduledAt: sched}
}

func (a *App) cmsArticleSave(w http.ResponseWriter, r *http.Request) {	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	art := articleFromForm(r, 0)
	if art.Status == "" {
		art.Status = cms.StatusDraft
	}
	if (art.Status == cms.StatusPublished || art.Status == cms.StatusScheduled) &&
		!rbac.Has(mw.RoleOf(r), rbac.PermWebPagesPublish) {
		art.Status = cms.StatusReview
	}
	id, err := a.web.SaveArticle(a.officeOf(r), art, a.uidPtr(r))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.RecordHistory(a.officeOf(r), "article", id, "create", "", art.Status, a.uid(r), "", clientIPOf(r))
	http.Redirect(w, r, "/cms/articles?flash=Artikel+disimpan", http.StatusSeeOther)
}

func (a *App) cmsArticleUpdate(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	var old string
	_ = a.db.QueryRow(`SELECT status FROM website_articles WHERE id=? AND office_id=? AND deleted_at IS NULL`, id, a.officeOf(r)).Scan(&old)
	if old == "" {
		http.NotFound(w, r)
		return
	}
	art := articleFromForm(r, id)
	if art.Status == "" {
		art.Status = old
	}
	if (art.Status == cms.StatusPublished || art.Status == cms.StatusScheduled) &&
		old != cms.StatusPublished && old != cms.StatusScheduled &&
		!rbac.Has(mw.RoleOf(r), rbac.PermWebPagesPublish) {
		http.Error(w, "butuh izin publikasi", http.StatusForbidden)
		return
	}
	if _, err := a.web.SaveArticle(a.officeOf(r), art, a.uidPtr(r)); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.RecordHistory(a.officeOf(r), "article", id, "update", old, art.Status, a.uid(r), "", clientIPOf(r))
	http.Redirect(w, r, "/cms/articles?flash=Artikel+disimpan", http.StatusSeeOther)
}

// ---- categories / services / faq / nav ----

func (a *App) cmsCategorySave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	_ = r.ParseForm()
	slug := strings.ToLower(strings.TrimSpace(r.FormValue("slug")))
	name := strings.TrimSpace(r.FormValue("name"))
	if name == "" || slug == "" {
		http.Error(w, "nama dan slug wajib", http.StatusBadRequest)
		return
	}
	r2, err := a.db.Exec(`INSERT INTO website_categories(office_id, name, slug) VALUES(?,?,?)
		ON DUPLICATE KEY UPDATE name=VALUES(name)`, a.officeOf(r), name, slug)
	_ = r2
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/articles?flash=Kategori+disimpan", http.StatusSeeOther)
}

func (a *App) cmsServiceSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	_ = r.ParseForm()
	pos, _ := strconv.Atoi(r.FormValue("position"))
	sv := &cms.Service{Name: strings.TrimSpace(r.FormValue("name")), Slug: strings.TrimSpace(r.FormValue("slug")),
		Description: r.FormValue("description"), Content: cms.SanitizeHTML(r.FormValue("content")),
		Position: pos, Visible: r.FormValue("visible") != "0",
		SEO: cms.SEO{Title: r.FormValue("seo_title"), Desc: r.FormValue("seo_desc")}}
	if v := r.FormValue("image_media_id"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			sv.ImageMediaID = sql.NullInt64{Int64: n, Valid: true}
		}
	}
	if _, err := a.web.SaveService(a.officeOf(r), sv); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/cms/services?flash=Layanan+disimpan", http.StatusSeeOther)
}

func (a *App) cmsServices(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	list, _ := a.web.PublicServices(a.officeOf(r))
	// Admin sees all incl. hidden: reload with visible flag.
	rows, _ := a.db.Query(`SELECT id, name, slug, description, position, visible FROM website_services WHERE office_id=? ORDER BY position, id`, a.officeOf(r))
	var out []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var n, s, d string
			var p, v int
			_ = rows.Scan(&id, &n, &s, &d, &p, &v)
			out = append(out, map[string]any{"ID": id, "Name": n, "Slug": s, "Desc": d, "Pos": p, "Vis": v == 1})
		}
	}
	_ = list
	a.render(w, r, "cms_services", "Layanan Website", map[string]any{"Rows": out})
}

func (a *App) cmsFAQ(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	rows, _ := a.db.Query(`SELECT id, category, question, position, visible FROM website_faqs WHERE office_id=? ORDER BY position, id`, a.officeOf(r))
	var out []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var c, q string
			var p, v int
			_ = rows.Scan(&id, &c, &q, &p, &v)
			out = append(out, map[string]any{"ID": id, "Cat": c, "Q": q, "Pos": p, "Vis": v == 1})
		}
	}
	a.render(w, r, "cms_faq", "FAQ Website", map[string]any{"Rows": out})
}

func (a *App) cmsFAQSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	_ = r.ParseForm()
	pos, _ := strconv.Atoi(r.FormValue("position"))
	f := &cms.FAQ{Category: r.FormValue("category"), Question: r.FormValue("question"),
		Answer: cms.SanitizeHTML(r.FormValue("answer")), Position: pos, Visible: r.FormValue("visible") != "0"}
	if _, err := a.web.SaveFAQ(a.officeOf(r), f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/cms/faq?flash=FAQ+disimpan", http.StatusSeeOther)
}

func (a *App) cmsFAQDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebArticles) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`DELETE FROM website_faqs WHERE id=? AND office_id=?`, id, a.officeOf(r))
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/faq?flash=FAQ+dihapus", http.StatusSeeOther)
}

func (a *App) cmsNav(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebNav) {
		return
	}
	rows, _ := a.db.Query(`SELECT id, area, label, url, parent_id, position, visible, new_tab FROM website_nav_items WHERE office_id=? ORDER BY area, position, id`, a.officeOf(r))
	var out []map[string]any
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var id int64
			var area, label, url string
			var par sql.NullInt64
			var p, v, t int
			_ = rows.Scan(&id, &area, &label, &url, &par, &p, &v, &t)
			pid := int64(0)
			if par.Valid {
				pid = par.Int64
			}
			out = append(out, map[string]any{"ID": id, "Area": area, "Label": label, "URL": url, "Parent": pid, "Pos": p, "Vis": v == 1, "Tab": t == 1})
		}
	}
	a.render(w, r, "cms_nav", "Navigasi Website", map[string]any{"Rows": out})
}

func (a *App) cmsNavSave(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebNav) {
		return
	}
	_ = r.ParseForm()
	pos, _ := strconv.Atoi(r.FormValue("position"))
	n := &cms.NavItem{Area: r.FormValue("area"), Label: r.FormValue("label"), URL: r.FormValue("url"),
		Position: pos, Visible: r.FormValue("visible") != "0", NewTab: r.FormValue("new_tab") == "1"}
	if v := r.FormValue("parent_id"); v != "" && v != "0" {
		if pi, err := strconv.ParseInt(v, 10, 64); err == nil {
			n.ParentID = sql.NullInt64{Int64: pi, Valid: true}
		}
	}
	if _, err := a.web.SaveNavItem(a.officeOf(r), n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/cms/nav?flash=Navigasi+disimpan", http.StatusSeeOther)
}

func (a *App) cmsNavDelete(w http.ResponseWriter, r *http.Request) {
	if !a.requireWeb(w, r, rbac.PermWebNav) {
		return
	}
	id, _ := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	_, _ = a.db.Exec(`DELETE FROM website_nav_items WHERE id=? AND office_id=?`, id, a.officeOf(r))
	a.web.Invalidate(a.officeOf(r))
	http.Redirect(w, r, "/cms/nav?flash=Menu+dihapus", http.StatusSeeOther)
}
