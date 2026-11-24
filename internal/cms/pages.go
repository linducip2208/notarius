package cms

import (
	"database/sql"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// Page statuses (workflow: DRAFT -> REVIEW -> PUBLISHED/SCHEDULED; ARCHIVED terminal).
const (
	StatusDraft     = "DRAFT"
	StatusReview    = "REVIEW"
	StatusScheduled = "SCHEDULED"
	StatusPublished = "PUBLISHED"
	StatusArchived  = "ARCHIVED"
)

// Page is a CMS page (relational, queryable — never one big JSON blob).
type Page struct {
	ID                       int64
	Title, Slug, Excerpt     string
	Content                  string
	FeaturedMediaID          sql.NullInt64
	Template                 string
	Status                   string
	Locale                   string
	SEO                      SEO
	CreatedBy, UpdatedBy     sql.NullInt64
	CreatedAt, UpdatedAt     time.Time
	PublishedAt, ScheduledAt sql.NullTime
}

// SEO holds per-document metadata (validated, no fake ratings).
type SEO struct {
	Title, Desc, Canonical, OGImage, Robots string
}

func validStatus(st string) bool {
	switch st {
	case StatusDraft, StatusReview, StatusScheduled, StatusPublished, StatusArchived:
		return true
	}
	return false
}

func cleanSlug(s string) (string, error) {
	s = strings.ToLower(strings.TrimSpace(s))
	if !slugRe.MatchString(s) {
		return "", errors.New("slug hanya [a-z0-9-]")
	}
	return s, nil
}

// cleanLocale allows only 'id' (default) or 'en'; anything else falls back.
func cleanLocale(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "en":
		return "en"
	default:
		return "id"
	}
}

// SavePage creates or updates a page (slug unique per office). Publish rules:
// PUBLISHED needs title+content; SCHEDULED needs future scheduled_at.
func (s *Store) SavePage(officeID int64, p *Page, actorID *int64) (int64, error) {
	slug, err := cleanSlug(p.Slug)
	if err != nil {
		return 0, err
	}
	if !validStatus(p.Status) {
		return 0, errors.New("status tidak dikenal")
	}
	if p.Title == "" {
		return 0, errors.New("judul wajib diisi")
	}
	if p.Status == StatusPublished && strings.TrimSpace(p.Content) == "" {
		return 0, errors.New("terbit butuh isi konten")
	}
	if p.Status == StatusScheduled {
		if !p.ScheduledAt.Valid || p.ScheduledAt.Time.Before(time.Now().Add(time.Minute)) {
			return 0, errors.New("terjadwal butuh waktu masa depan")
		}
	}
	p.Slug = slug
	p.Locale = cleanLocale(p.Locale)
	if p.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO website_pages(office_id, title, slug, excerpt, content,
			featured_media_id, template, status, scheduled_at, seo_title, seo_desc, canonical_url,
			og_image, robots, created_by, updated_by, locale) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			officeID, p.Title, p.Slug, p.Excerpt, p.Content, nullInt(p.FeaturedMediaID),
			p.Template, p.Status, nullTime(p.ScheduledAt), p.SEO.Title, p.SEO.Desc,
			p.SEO.Canonical, p.SEO.OGImage, seoRobots(p.SEO.Robots), nullInt64(actorID), nullInt64(actorID), p.Locale)
		if err != nil {
			return 0, dupErr(err, "slug sudah dipakai")
		}
		id, _ := res.LastInsertId()
		s.Invalidate(officeID)
		return id, nil
	}
	var old string
	_ = s.db.QueryRow(`SELECT status FROM website_pages WHERE id=? AND office_id=? AND deleted_at IS NULL`,
		p.ID, officeID).Scan(&old)
	if old == "" {
		return 0, errors.New("halaman tidak ditemukan")
	}
	if _, err := s.db.Exec(`UPDATE website_pages SET title=?, slug=?, excerpt=?, content=?,
		featured_media_id=?, template=?, status=?, scheduled_at=?, seo_title=?, seo_desc=?,
		canonical_url=?, og_image=?, robots=?, updated_by=?, locale=? WHERE id=? AND office_id=?`,
		p.Title, p.Slug, p.Excerpt, p.Content, nullInt(p.FeaturedMediaID), p.Template,
		p.Status, nullTime(p.ScheduledAt), p.SEO.Title, p.SEO.Desc, p.SEO.Canonical,
		p.SEO.OGImage, seoRobots(p.SEO.Robots), nullInt64(actorID), p.Locale, p.ID, officeID); err != nil {
		return 0, dupErr(err, "slug sudah dipakai")
	}
	s.Invalidate(officeID)
	return p.ID, nil
}

// PublishDue flips due SCHEDULED rows to PUBLISHED (called from the daily scan).
func (s *Store) PublishDue(officeID int64) (pages, articles int64) {
	r, _ := s.db.Exec(`UPDATE website_pages SET status='PUBLISHED', published_at=UTC_TIMESTAMP()
		WHERE office_id=? AND status='SCHEDULED' AND scheduled_at <= UTC_TIMESTAMP()`, officeID)
	if r != nil {
		pages, _ = r.RowsAffected()
	}
	r2, _ := s.db.Exec(`UPDATE website_articles SET status='PUBLISHED', published_at=UTC_TIMESTAMP()
		WHERE office_id=? AND status='SCHEDULED' AND scheduled_at <= UTC_TIMESTAMP()`, officeID)
	if r2 != nil {
		articles, _ = r2.RowsAffected()
	}
	if pages+articles > 0 {
		s.Invalidate(officeID)
	}
	return pages, articles
}

func seoRobots(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "noindex,nofollow" || v == "noindex,follow" || v == "index,nofollow" {
		return v
	}
	return "index,follow"
}

func dupErr(err error, msg string) error {
	if err != nil && strings.Contains(strings.ToLower(err.Error()), "duplicate") {
		return errors.New(msg)
	}
	return err
}

func nullInt(v sql.NullInt64) any {
	if v.Valid {
		return v.Int64
	}
	return nil
}

func nullStr(v string) any {
	if strings.TrimSpace(v) == "" {
		return nil
	}
	return v
}

func nullInt64(p *int64) any {
	if p != nil {
		return *p
	}
	return nil
}

func nullTime(v sql.NullTime) any {
	if v.Valid {
		return v.Time
	}
	return nil
}

// Section types and their validated settings schema.
var sectionTypes = map[string]bool{
	"hero": true, "intro": true, "profile": true, "services": true, "process": true,
	"info": true, "articles": true, "faq": true, "contact": true, "map": true,
	"footer": true, "text": true, "gallery": true, "cta": true,
}

var sectionBG = map[string]bool{"light": true, "white": true, "dark": true, "primary": true}
var sectionAlign = map[string]bool{"left": true, "center": true, "right": true}
var sectionLayout = map[string]bool{"standard": true, "wide": true, "split": true}

// Section is one homepage/page-builder block with validated settings JSON.
type Section struct {
	ID                        int64
	PageID                    int64
	Type                      string
	Title, Subtitle, Content  string
	ImageMediaID              sql.NullInt64
	CTALabel, CTAURL          string
	Background, Align, Layout string
	Visible                   bool
	Position                  int
	Settings                  map[string]any
}

// ValidateSection enforces type/background/align/layout enums, CTA URL safety,
// and a strict settings schema (columns 1-3, ratio, theme).
func ValidateSection(sec *Section) error {
	if !sectionTypes[sec.Type] {
		return errors.New("tipe section tidak dikenal")
	}
	if !sectionBG[sec.Background] {
		sec.Background = "light"
	}
	if !sectionAlign[sec.Align] {
		sec.Align = "left"
	}
	if !sectionLayout[sec.Layout] {
		sec.Layout = "standard"
	}
	if sec.CTALabel != "" && !ValidNavURL(sec.CTAURL) {
		return errors.New("URL CTA tidak valid/aman")
	}
	out := map[string]any{}
	for k, v := range sec.Settings {
		switch k {
		case "columns":
			switch n := v.(type) {
			case float64:
				if n >= 1 && n <= 3 {
					out[k] = int(n)
				}
			case int:
				if n >= 1 && n <= 3 {
					out[k] = n
				}
			}
		case "ratio":
			if s, ok := v.(string); ok && (s == "16:9" || s == "4:3" || s == "1:1") {
				out[k] = s
			}
		case "theme":
			if s, ok := v.(string); ok && (s == "auto" || s == "light" || s == "dark") {
				out[k] = s
			}
		}
	}
	sec.Settings = out
	return nil
}

// SaveSection inserts or updates one section (validated first).
func (s *Store) SaveSection(officeID int64, sec *Section) (int64, error) {
	if err := ValidateSection(sec); err != nil {
		return 0, err
	}
	var raw []byte
	if len(sec.Settings) > 0 {
		raw, _ = json.Marshal(sec.Settings)
	}
	vis := 0
	if sec.Visible {
		vis = 1
	}
	if sec.ID == 0 {
		var pid int64
		if err := s.db.QueryRow(`SELECT office_id FROM website_pages WHERE id=? AND deleted_at IS NULL`,
			sec.PageID).Scan(&pid); err != nil || pid != officeID {
			return 0, errors.New("halaman tidak ditemukan di kantor ini")
		}
		res, err := s.db.Exec(`INSERT INTO website_page_sections(page_id, office_id, stype, title,
			subtitle, content, image_media_id, cta_label, cta_url, background, align, layout,
			visible, position, settings) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			sec.PageID, officeID, sec.Type, sec.Title, sec.Subtitle, sec.Content,
			nullInt(sec.ImageMediaID), sec.CTALabel, sec.CTAURL, sec.Background, sec.Align,
			sec.Layout, vis, sec.Position, nullJSON(raw))
		if err != nil {
			return 0, err
		}
		id, _ := res.LastInsertId()
		s.Invalidate(officeID)
		return id, nil
	}
	if _, err := s.db.Exec(`UPDATE website_page_sections SET title=?, subtitle=?, content=?,
		image_media_id=?, cta_label=?, cta_url=?, background=?, align=?, layout=?, visible=?,
		position=?, settings=? WHERE id=? AND office_id=?`,
		sec.Title, sec.Subtitle, sec.Content, nullInt(sec.ImageMediaID), sec.CTALabel,
		sec.CTAURL, sec.Background, sec.Align, sec.Layout, vis, sec.Position,
		nullJSON(raw), sec.ID, officeID); err != nil {
		return 0, err
	}
	s.Invalidate(officeID)
	return sec.ID, nil
}

func nullJSON(raw []byte) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}
