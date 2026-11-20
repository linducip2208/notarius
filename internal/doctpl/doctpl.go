package doctpl

import (
	"database/sql"
	"errors"

	"notaryos/internal/docengine"
)

// Template is an approved-versioned document template.
type Template struct {
	ID       int64
	Name     string
	Category string
	Body     string
	Version  int
	Approved bool
}

// Clause is a reusable approved clause.
type Clause struct {
	ID       int64
	Title    string
	Body     string
	Tags     string
	Approved bool
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Save creates v1 or a new version of a template. Approval resets on edit.
func (s *Store) Save(officeID int64, id int64, name, category, body string) (int64, error) {
	if name == "" || body == "" {
		return 0, errors.New("nama dan isi template wajib diisi")
	}
	if id == 0 {
		res, err := s.db.Exec(`INSERT INTO doc_templates(office_id, name, category, body) VALUES(?,?,?,?)`,
			officeID, name, category, body)
		if err != nil {
			return 0, err
		}
		return res.LastInsertId()
	}
	var owner int64
	if err := s.db.QueryRow(`SELECT office_id FROM doc_templates WHERE id=?`, id).Scan(&owner); err != nil {
		return 0, err
	}
	if owner != officeID {
		return 0, errors.New("template milik kantor lain")
	}
	_, err := s.db.Exec(`UPDATE doc_templates SET name=?, category=?, body=?, version=version+1,
		approved=0, approved_by=NULL WHERE id=?`, name, category, body, id)
	return id, err
}

// Approve marks a template as notary-approved for deed generation.
func (s *Store) Approve(id, approverID int64) error {
	_, err := s.db.Exec(`UPDATE doc_templates SET approved=1, approved_by=? WHERE id=?`, approverID, id)
	return err
}

// Get loads one template scoped to office.
func (s *Store) Get(officeID, id int64) (Template, error) {
	var t Template
	err := s.db.QueryRow(`SELECT id, name, category, body, version, approved FROM doc_templates
		WHERE id=? AND office_id=?`, id, officeID).
		Scan(&t.ID, &t.Name, &t.Category, &t.Body, &t.Version, &t.Approved)
	return t, err
}

// List returns all templates of the office.
func (s *Store) List(officeID int64) ([]Template, error) {
	rows, err := s.db.Query(`SELECT id, name, category, version, approved FROM doc_templates WHERE office_id=? ORDER BY name`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Template
	for rows.Next() {
		var t Template
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Version, &t.Approved); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// Preview renders a template with vars + draft watermark; missing vars are listed.
func Preview(body string, vars map[string]string) (rendered string, missing []string) {
	missing = docengine.Missing(body, vars)
	return docengine.Watermark(docengine.Render(body, vars)), missing
}

// SaveClause stores a clause (approval by notary required before use in finals).
func (s *Store) SaveClause(officeID int64, title, body, tags string) (int64, error) {
	if title == "" || body == "" {
		return 0, errors.New("judul dan isi klausul wajib diisi")
	}
	res, err := s.db.Exec(`INSERT INTO clauses(office_id, title, body, tags) VALUES(?,?,?,?)`,
		officeID, title, body, tags)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Clauses lists office clauses.
func (s *Store) Clauses(officeID int64) ([]Clause, error) {
	rows, err := s.db.Query(`SELECT id, title, body, tags, approved FROM clauses WHERE office_id=? ORDER BY title`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Clause
	for rows.Next() {
		var c Clause
		var tags sql.NullString
		if err := rows.Scan(&c.ID, &c.Title, &c.Body, &tags, &c.Approved); err != nil {
			return nil, err
		}
		if tags.Valid {
			c.Tags = tags.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}
