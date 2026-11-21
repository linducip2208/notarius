package cms

import (
	"database/sql"
	"encoding/json"
	"html"
	"strings"
)

func jsonUnmarshal(raw string, v *map[string]any) {
	*v = map[string]any{}
	if raw == "" || raw == "{}" {
		return
	}
	_ = json.Unmarshal([]byte(raw), v)
}

// SanitizeHTML allows a strict tag set for CMS content.
// a[href] and img[src] URLs are validated (relative or http(s) only);
// all event handlers, scripts, iframes, styles are dropped.
func SanitizeHTML(in string) string {
	allowed := map[string]bool{
		"p": true, "br": true, "b": true, "strong": true, "i": true, "em": true,
		"u": true, "ul": true, "ol": true, "li": true, "h2": true, "h3": true,
		"h4": true, "blockquote": true, "a": true, "img": true,
	}
	voidTags := map[string]bool{"br": true, "img": true}
	var sb strings.Builder
	i := 0
	openA := 0
	for i < len(in) {
		if in[i] != '<' {
			sb.WriteString(html.EscapeString(in[i : i+1]))
			i++
			continue
		}
		end := strings.IndexByte(in[i:], '>')
		if end < 0 {
			sb.WriteString(html.EscapeString(in[i:]))
			break
		}
		raw := strings.TrimSpace(in[i+1 : i+end])
		if strings.Contains(raw, "<") {
			sb.WriteString("&lt;")
			i++
			continue
		}
		closing := false
		if strings.HasPrefix(raw, "/") {
			closing = true
			raw = strings.TrimSpace(raw[1:])
		}
		name, attrs := splitTag(raw)
		name = strings.ToLower(strings.TrimSuffix(name, "/"))
		if !allowed[name] {
			i += end + 1
			continue
		}
		if closing {
			if voidTags[name] {
				i += end + 1
				continue
			}
			if name == "a" {
				if openA == 0 {
					i += end + 1
					continue
				}
				openA--
			}
			sb.WriteString("</" + name + ">")
			i += end + 1
			continue
		}
		switch name {
		case "a":
			href := tagAttr(attrs, "href")
			if !ValidNavURL(href) {
				i += end + 1
				continue
			}
			openA++
			sb.WriteString(`<a href="` + html.EscapeString(href) + `">`)
		case "img":
			src := tagAttr(attrs, "src")
			if !ValidNavURL(src) && !strings.HasPrefix(src, "/media/") {
				i += end + 1
				continue
			}
			alt := tagAttr(attrs, "alt")
			sb.WriteString(`<img src="` + html.EscapeString(src) + `" alt="` + html.EscapeString(alt) + `" loading="lazy">`)
		case "br":
			sb.WriteString("<br>")
		default:
			sb.WriteString("<" + name + ">")
		}
		i += end + 1
	}
	return sb.String()
}

func splitTag(raw string) (string, string) {
	for i, ch := range raw {
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '/' {
			return raw[:i], raw[i:]
		}
	}
	return raw, ""
}

// tagAttr extracts a single attribute value (double/single quoted or bare).
func tagAttr(attrs, key string) string {
	lower := strings.ToLower(attrs)
	idx := strings.Index(lower, key+"=")
	if idx < 0 {
		return ""
	}
	rest := strings.TrimSpace(attrs[idx+len(key)+1:])
	if rest == "" {
		return ""
	}
	if rest[0] == '"' || rest[0] == '\'' {
		q := rest[0]
		if j := strings.IndexByte(rest[1:], q); j >= 0 {
			return rest[1 : 1+j]
		}
		return ""
	}
	j := strings.IndexAny(rest, " \t\n/>")
	if j < 0 {
		return rest
	}
	return rest[:j]
}

// ---- public reads (published only, office-scoped) ----

// PublicPage returns a PUBLISHED non-deleted page by slug (+ visible sections).
// Locale falls back to 'id' when the requested translation is missing.
func (s *Store) PublicPage(officeID int64, slug, locale string) (Page, []Section, error) {
	locale = cleanLocale(locale)
	for _, loc := range []string{locale, "id"} {
		if p, secs, err := s.publicPageLoc(officeID, slug, loc); err == nil {
			return p, secs, nil
		}
	}
	return Page{}, nil, sql.ErrNoRows
}

func (s *Store) publicPageLoc(officeID int64, slug, locale string) (Page, []Section, error) {
	var p Page
	var feat sql.NullInt64
	var sched, pub sql.NullTime
	err := s.db.QueryRow(`SELECT id, title, slug, excerpt, content, featured_media_id,
		template, status, seo_title, seo_desc, canonical_url, og_image, robots,
		created_at, scheduled_at, published_at
		FROM website_pages WHERE office_id=? AND slug=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL`,
		officeID, slug, locale).Scan(&p.ID, &p.Title, &p.Slug, &p.Excerpt, &p.Content, &feat,
		&p.Template, &p.Status, &p.SEO.Title, &p.SEO.Desc, &p.SEO.Canonical, &p.SEO.OGImage,
		&p.SEO.Robots, &p.CreatedAt, &sched, &pub)
	if err != nil {
		return Page{}, nil, err
	}
	p.FeaturedMediaID = feat
	p.ScheduledAt, p.PublishedAt = sched, pub
	secs, _ := s.SectionsOf(p.ID, true)
	return p, secs, nil
}

// SectionsOf lists sections in position order (optionally visible-only).
func (s *Store) SectionsOf(pageID int64, visibleOnly bool) ([]Section, error) {
	q := `SELECT id, page_id, stype, title, subtitle, content, image_media_id, cta_label,
		cta_url, background, align, layout, visible, position, COALESCE(settings,'{}')
		FROM website_page_sections WHERE page_id=?`
	if visibleOnly {
		q += ` AND visible=1`
	}
	q += ` ORDER BY position, id`
	rows, err := s.db.Query(q, pageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Section
	for rows.Next() {
		var sec Section
		var raw string
		var vis int
		_ = rows.Scan(&sec.ID, &sec.PageID, &sec.Type, &sec.Title, &sec.Subtitle,
			&sec.Content, &sec.ImageMediaID, &sec.CTALabel, &sec.CTAURL, &sec.Background,
			&sec.Align, &sec.Layout, &vis, &sec.Position, &raw)
		sec.Visible = vis == 1
		sec.Settings = map[string]any{}
		jsonUnmarshal(raw, &sec.Settings)
		out = append(out, sec)
	}
	return out, rows.Err()
}

// PublicArticles lists published articles newest first (search + pagination).
// Empty 'en' result falls back to 'id' so untranslated offices never look empty.
func (s *Store) PublicArticles(officeID int64, locale, q string, limit, offset int) ([]Article, int, error) {
	locale = cleanLocale(locale)
	list, total, err := s.publicArticlesLoc(officeID, locale, q, limit, offset)
	if err != nil || (locale == "en" && total == 0 && strings.TrimSpace(q) == "") {
		if fb, ft, ferr := s.publicArticlesLoc(officeID, "id", q, limit, offset); ferr == nil && (err != nil || ft > 0) {
			return fb, ft, ferr
		}
	}
	return list, total, err
}

func (s *Store) publicArticlesLoc(officeID int64, locale, q string, limit, offset int) ([]Article, int, error) {
	where := `office_id=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL`
	args := []any{officeID, locale}
	if strings.TrimSpace(q) != "" {
		where += ` AND (title LIKE ? OR excerpt LIKE ? OR tags LIKE ?)`
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, like)
	}
	var total int
	_ = s.db.QueryRow(`SELECT COUNT(*) FROM website_articles WHERE `+where, args...).Scan(&total)
	rows, err := s.db.Query(`SELECT id, category_id, title, slug, excerpt, cover_media_id,
		tags, author, published_at FROM website_articles WHERE `+where+`
		ORDER BY published_at DESC, id DESC LIMIT ? OFFSET ?`, append(args, limit, offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var out []Article
	for rows.Next() {
		var a Article
		_ = rows.Scan(&a.ID, &a.CategoryID, &a.Title, &a.Slug, &a.Excerpt, &a.CoverMediaID,
			&a.Tags, &a.Author, &a.PublishedAt)
		out = append(out, a)
	}
	return out, total, rows.Err()
}

// PublicArticle returns one published article + related (same category).
func (s *Store) PublicArticle(officeID int64, slug, locale string) (Article, []Article, error) {
	locale = cleanLocale(locale)
	for _, loc := range []string{locale, "id"} {
		if a, rel, err := s.publicArticleLoc(officeID, slug, loc); err == nil {
			return a, rel, nil
		}
	}
	return Article{}, nil, sql.ErrNoRows
}

func (s *Store) publicArticleLoc(officeID int64, slug, locale string) (Article, []Article, error) {
	var a Article
	var cat sql.NullInt64
	var sched, pub sql.NullTime
	err := s.db.QueryRow(`SELECT id, category_id, title, slug, excerpt, content, cover_media_id,
		tags, author, status, seo_title, seo_desc, canonical_url, og_image, robots,
		created_at, scheduled_at, published_at
		FROM website_articles WHERE office_id=? AND slug=? AND locale=? AND status='PUBLISHED' AND deleted_at IS NULL`,
		officeID, slug, locale).Scan(&a.ID, &cat, &a.Title, &a.Slug, &a.Excerpt, &a.Content,
		&a.CoverMediaID, &a.Tags, &a.Author, &a.Status, &a.SEO.Title, &a.SEO.Desc,
		&a.SEO.Canonical, &a.SEO.OGImage, &a.SEO.Robots, &a.CreatedAt, &sched, &pub)
	if err != nil {
		return Article{}, nil, err
	}
	a.CategoryID, a.ScheduledAt, a.PublishedAt = cat, sched, pub
	var rel []Article
	if cat.Valid {
		rows, _ := s.db.Query(`SELECT id, title, slug, excerpt, published_at FROM website_articles
			WHERE office_id=? AND category_id=? AND status='PUBLISHED' AND id<>? AND deleted_at IS NULL
			ORDER BY published_at DESC LIMIT 3`, officeID, cat.Int64, a.ID)
		if rows != nil {
			defer rows.Close()
			for rows.Next() {
				var r Article
				_ = rows.Scan(&r.ID, &r.Title, &r.Slug, &r.Excerpt, &r.PublishedAt)
				rel = append(rel, r)
			}
		}
	}
	return a, rel, nil
}

// PublicFAQs lists visible FAQs ordered (locale with 'id' fallback).
func (s *Store) PublicFAQs(officeID int64, locale string) ([]FAQ, error) {
	locale = cleanLocale(locale)
	if out, err := s.publicFAQsLoc(officeID, locale); err == nil && (locale == "id" || len(out) > 0) {
		return out, err
	}
	return s.publicFAQsLoc(officeID, "id")
}

func (s *Store) publicFAQsLoc(officeID int64, locale string) ([]FAQ, error) {
	rows, err := s.db.Query(`SELECT id, category, question, answer, position FROM website_faqs
		WHERE office_id=? AND locale=? AND visible=1 ORDER BY position, id`, officeID, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FAQ
	for rows.Next() {
		var f FAQ
		_ = rows.Scan(&f.ID, &f.Category, &f.Question, &f.Answer, &f.Position)
		out = append(out, f)
	}
	return out, rows.Err()
}

// PublicNav lists visible items for an area (flat; parent grouping in template/handler).
func (s *Store) PublicNav(officeID int64, area string) ([]NavItem, error) {
	if area != "header" && area != "footer" {
		area = "header"
	}
	rows, err := s.db.Query(`SELECT id, label, url, parent_id, position, new_tab FROM website_nav_items
		WHERE office_id=? AND area=? AND visible=1 ORDER BY position, id`, officeID, area)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []NavItem
	for rows.Next() {
		var n NavItem
		var tab int
		_ = rows.Scan(&n.ID, &n.Label, &n.URL, &n.ParentID, &n.Position, &tab)
		n.NewTab = tab == 1
		out = append(out, n)
	}
	return out, rows.Err()
}

// PublicServices lists visible service info entries (locale with 'id' fallback).
func (s *Store) PublicServices(officeID int64, locale string) ([]Service, error) {
	locale = cleanLocale(locale)
	if out, err := s.publicServicesLoc(officeID, locale); err == nil && (locale == "id" || len(out) > 0) {
		return out, err
	}
	return s.publicServicesLoc(officeID, "id")
}

func (s *Store) publicServicesLoc(officeID int64, locale string) ([]Service, error) {
	rows, err := s.db.Query(`SELECT id, name, slug, description, content, image_media_id,
		position, seo_title, seo_desc, service_code FROM website_services
		WHERE office_id=? AND locale=? AND visible=1 ORDER BY position, id`, officeID, locale)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Service
	for rows.Next() {
		var sv Service
		var code sql.NullString
		_ = rows.Scan(&sv.ID, &sv.Name, &sv.Slug, &sv.Description, &sv.Content,
			&sv.ImageMediaID, &sv.Position, &sv.SEO.Title, &sv.SEO.Desc, &code)
		if code.Valid {
			sv.ServiceCode = code.String
		}
		out = append(out, sv)
	}
	return out, rows.Err()
}

// SitemapURLs returns published slugs for sitemap (pages + articles only).
func (s *Store) SitemapURLs(officeID int64) (pages, articles []string) {
	pr, _ := s.db.Query(`SELECT slug FROM website_pages WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL`, officeID)
	if pr != nil {
		defer pr.Close()
		for pr.Next() {
			var sl string
			_ = pr.Scan(&sl)
			pages = append(pages, sl)
		}
	}
	ar, _ := s.db.Query(`SELECT slug FROM website_articles WHERE office_id=? AND status='PUBLISHED' AND deleted_at IS NULL`, officeID)
	if ar != nil {
		defer ar.Close()
		for ar.Next() {
			var sl string
			_ = ar.Scan(&sl)
			articles = append(articles, sl)
		}
	}
	return pages, articles
}
