package cms

import (
	"database/sql"
	"errors"
	"strings"
	"time"
)

// ---- articles + categories ----

// Article is a CMS article with workflow status.
type Article struct {
	ID                       int64
	CategoryID               sql.NullInt64
	Title, Slug, Excerpt     string
	Content                  string
	CoverMediaID             sql.NullInt64
	Tags, Author             string
	Status                   string
	SEO                      SEO
	CreatedBy, UpdatedBy     sql.NullInt64
	CreatedAt                time.Time
	PublishedAt, ScheduledAt sql.NullTime
}

// SaveArticle creates/updates with slug uniqueness + publish rules.
func (s *Store) SaveArticle(officeID int64, a *Article, actorID *int64) (int64, error) {
	slug, err := cleanSlug(a.Slug)
	if err != nil {
		return 0, err
	}
	if !validStatus(a.Status) {
		return 0, errors.New("status tidak dikenal")
	}
	if a.Title == "" || strings.TrimSpace(a.Content) == "" {
		return 0, errors.New("judul dan isi wajib diisi")
	}
	if a.Status == StatusPublished && strings.TrimSpace(a.Content) == "" {
		return 0, errors.New("terbit butuh isi konten")
	}
	a.Slug = slug
	if a.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO website_articles(office_id, category_id, title, slug,
			excerpt, content, cover_media_id, tags, author, status, scheduled_at, seo_title,
			seo_desc, canonical_url, og_image, robots, created_by, updated_by)
			VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
			officeID, nullInt(a.CategoryID), a.Title, a.Slug, a.Excerpt, a.Content,
			nullInt(a.CoverMediaID), a.Tags, a.Author, a.Status, nullTime(a.ScheduledAt),
			a.SEO.Title, a.SEO.Desc, a.SEO.Canonical, a.SEO.OGImage, seoRobots(a.SEO.Robots),
			nullInt64(actorID), nullInt64(actorID))
		if err != nil {
			return 0, dupErr(err, "slug sudah dipakai")
		}
		id, _ := res.LastInsertId()
		s.Invalidate(officeID)
		return id, nil
	}
	var old string
	_ = s.db.QueryRow(`SELECT status FROM website_articles WHERE id=? AND office_id=? AND deleted_at IS NULL`,
		a.ID, officeID).Scan(&old)
	if old == "" {
		return 0, errors.New("artikel tidak ditemukan")
	}
	if _, err := s.db.Exec(`UPDATE website_articles SET category_id=?, title=?, slug=?, excerpt=?,
		content=?, cover_media_id=?, tags=?, author=?, status=?, scheduled_at=?, seo_title=?,
		seo_desc=?, canonical_url=?, og_image=?, robots=?, updated_by=? WHERE id=? AND office_id=?`,
		nullInt(a.CategoryID), a.Title, a.Slug, a.Excerpt, a.Content, nullInt(a.CoverMediaID),
		a.Tags, a.Author, a.Status, nullTime(a.ScheduledAt), a.SEO.Title, a.SEO.Desc,
		a.SEO.Canonical, a.SEO.OGImage, seoRobots(a.SEO.Robots), nullInt64(actorID), a.ID, officeID); err != nil {
		return 0, dupErr(err, "slug sudah dipakai")
	}
	s.Invalidate(officeID)
	return a.ID, nil
}

// ---- services (marketing info pages, Notaris scope only) ----

// Service is a public service info entry (not the operational `services` table).
type Service struct {
	ID                       int64
	Name, Slug, Description  string
	Content                  string
	ImageMediaID             sql.NullInt64
	Position                 int
	Visible                  bool
	SEO                      SEO
}

// SaveService upserts a service info entry.
func (s *Store) SaveService(officeID int64, sv *Service) (int64, error) {
	slug, err := cleanSlug(sv.Slug)
	if err != nil {
		return 0, err
	}
	if sv.Name == "" {
		return 0, errors.New("nama layanan wajib diisi")
	}
	sv.Slug = slug
	vis := 0
	if sv.Visible {
		vis = 1
	}
	if sv.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO website_services(office_id, name, slug, description,
			content, image_media_id, position, visible, seo_title, seo_desc)
			VALUES(?,?,?,?,?,?,?,?,?,?)`, officeID, sv.Name, sv.Slug, sv.Description,
			sv.Content, nullInt(sv.ImageMediaID), sv.Position, vis, sv.SEO.Title, sv.SEO.Desc)
		if err != nil {
			return 0, dupErr(err, "slug sudah dipakai")
		}
		id, _ := res.LastInsertId()
		s.Invalidate(officeID)
		return id, nil
	}
	if _, err := s.db.Exec(`UPDATE website_services SET name=?, slug=?, description=?, content=?,
		image_media_id=?, position=?, visible=?, seo_title=?, seo_desc=? WHERE id=? AND office_id=?`,
		sv.Name, sv.Slug, sv.Description, sv.Content, nullInt(sv.ImageMediaID), sv.Position,
		vis, sv.SEO.Title, sv.SEO.Desc, sv.ID, officeID); err != nil {
		return 0, dupErr(err, "slug sudah dipakai")
	}
	s.Invalidate(officeID)
	return sv.ID, nil
}

// ---- FAQ ----

// FAQ is one public question/answer (only visible ones render publicly).
type FAQ struct {
	ID                 int64
	Category           string
	Question, Answer   string
	Position           int
	Visible            bool
}

// SaveFAQ upserts an FAQ entry.
func (s *Store) SaveFAQ(officeID int64, f *FAQ) (int64, error) {
	if strings.TrimSpace(f.Question) == "" || strings.TrimSpace(f.Answer) == "" {
		return 0, errors.New("pertanyaan dan jawaban wajib diisi")
	}
	if f.Category == "" {
		f.Category = "Umum"
	}
	vis := 0
	if f.Visible {
		vis = 1
	}
	if f.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO website_faqs(office_id, category, question, answer,
			position, visible) VALUES(?,?,?,?,?,?)`, officeID, f.Category, f.Question,
			f.Answer, f.Position, vis)
		if err != nil {
			return 0, err
		}
		id, _ := res.LastInsertId()
		s.Invalidate(officeID)
		return id, nil
	}
	if _, err := s.db.Exec(`UPDATE website_faqs SET category=?, question=?, answer=?,
		position=?, visible=? WHERE id=? AND office_id=?`, f.Category, f.Question,
		f.Answer, f.Position, vis, f.ID, officeID); err != nil {
		return 0, err
	}
	s.Invalidate(officeID)
	return f.ID, nil
}

// ---- navigation ----

// NavItem is one header/footer menu entry. URL validated (no javascript:).
type NavItem struct {
	ID              int64
	Area            string
	Label, URL      string
	ParentID        sql.NullInt64
	Position        int
	Visible, NewTab bool
}

// SaveNavItem upserts a navigation entry (URL safety enforced).
func (s *Store) SaveNavItem(officeID int64, n *NavItem) (int64, error) {
	if n.Area != "header" && n.Area != "footer" {
		n.Area = "header"
	}
	if strings.TrimSpace(n.Label) == "" || !ValidNavURL(n.URL) {
		return 0, errors.New("label dan URL aman wajib diisi")
	}
	if n.ParentID.Valid {
		var po int64
		if err := s.db.QueryRow(`SELECT office_id FROM website_nav_items WHERE id=?`,
			n.ParentID.Int64).Scan(&po); err != nil || po != officeID {
			return 0, errors.New("induk menu tidak valid")
		}
	}
	vis, tab := 0, 0
	if n.Visible {
		vis = 1
	}
	if n.NewTab {
		tab = 1
	}
	if n.ID == 0 {
		res, err := s.db.Exec(`INSERT INTO website_nav_items(office_id, area, label, url,
			parent_id, position, visible, new_tab) VALUES(?,?,?,?,?,?,?,?)`,
			officeID, n.Area, n.Label, strings.TrimSpace(n.URL), nullInt(n.ParentID),
			n.Position, vis, tab)
		if err != nil {
			return 0, err
		}
		id, _ := res.LastInsertId()
		s.Invalidate(officeID)
		return id, nil
	}
	if _, err := s.db.Exec(`UPDATE website_nav_items SET area=?, label=?, url=?, parent_id=?,
		position=?, visible=?, new_tab=? WHERE id=? AND office_id=?`, n.Area, n.Label,
		strings.TrimSpace(n.URL), nullInt(n.ParentID), n.Position, vis, tab, n.ID, officeID); err != nil {
		return 0, err
	}
	s.Invalidate(officeID)
	return n.ID, nil
}

// ---- publication history ----

// RecordHistory logs a publish workflow transition (actor, from→to, note, IP).
func (s *Store) RecordHistory(officeID int64, kind string, refID int64, action, from, to string, actorID *int64, note, ip string) {
	if kind != "page" && kind != "article" {
		return
	}
	_, _ = s.db.Exec(`INSERT INTO website_publication_history(office_id, kind, ref_id, action,
		from_status, to_status, actor_id, note, ip) VALUES(?,?,?,?,?,?,?,?,?)`,
		officeID, kind, refID, action, from, to, nullInt64(actorID), note, ip)
}

// History lists publication events newest first.
func (s *Store) History(officeID int64, limit int) ([]map[string]any, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(`SELECT h.kind, h.ref_id, h.action, h.from_status, h.to_status,
		COALESCE(u.name,''), h.note, h.ip, h.created_at
		FROM website_publication_history h LEFT JOIN users u ON u.id=h.actor_id
		WHERE h.office_id=? ORDER BY h.id DESC LIMIT ?`, officeID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var m = map[string]any{}
		var kind, action, from, to, actor, note, ip, at string
		var ref int64
		_ = rows.Scan(&kind, &ref, &action, &from, &to, &actor, &note, &ip, &at)
		m["Kind"], m["Ref"], m["Action"], m["From"], m["To"] = kind, ref, action, from, to
		m["Actor"], m["Note"], m["IP"], m["At"] = actor, note, ip, at
		out = append(out, m)
	}
	return out, rows.Err()
}
