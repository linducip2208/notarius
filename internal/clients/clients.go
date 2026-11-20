package clients

import (
	"database/sql"
	"strings"

	"notaryos/internal/validate"
)

// Client is an individual or corporate client of the office.
type Client struct {
	ID       int64
	OfficeID int64
	Kind     string // individual | corporate
	FullName string
	Email    string
	Phone    string
	Address  string
	NPWP     string
}

// DuplicateCandidate is a possible duplicate found by name/phone match.
type DuplicateCandidate struct {
	ID       int64
	FullName string
	Phone    string
	Reason   string
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

func (s *Store) Create(c Client) (int64, error) {
	if c.Kind != "individual" && c.Kind != "corporate" {
		return 0, errf("jenis klien harus individual atau corporate")
	}
	if err := validate.Required("nama", c.FullName); err != nil {
		return 0, err
	}
	if err := validate.Phone(c.Phone); err != nil {
		return 0, err
	}
	if c.Email != "" {
		if err := validate.Email(c.Email); err != nil {
			return 0, err
		}
	}
	if err := validate.NPWP(c.NPWP); err != nil {
		return 0, err
	}
	res, err := s.db.Exec(`INSERT INTO clients(office_id, kind, full_name, email, phone, address, npwp)
		VALUES(?,?,?,?,?,?,?)`, c.OfficeID, c.Kind, c.FullName, null(c.Email), c.Phone, null(c.Address), null(c.NPWP))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// FindDuplicates detects likely duplicates by exact phone or name similarity.
func (s *Store) FindDuplicates(officeID int64, name, phone string) ([]DuplicateCandidate, error) {
	var out []DuplicateCandidate
	rows, err := s.db.Query(`SELECT id, full_name, phone FROM clients
		WHERE office_id=? AND (phone=? OR full_name SOUNDS LIKE ?) LIMIT 10`, officeID, phone, name)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var d DuplicateCandidate
		if err := rows.Scan(&d.ID, &d.FullName, &d.Phone); err != nil {
			return nil, err
		}
		if d.Phone == phone {
			d.Reason = "nomor telepon sama"
		} else {
			d.Reason = "nama mirip"
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// List returns clients with optional search, newest first.
func (s *Store) List(officeID int64, q string, limit int) ([]Client, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	query := `SELECT id, office_id, kind, full_name, email, phone, address, npwp FROM clients WHERE office_id=?`
	args := []any{officeID}
	if strings.TrimSpace(q) != "" {
		query += ` AND (full_name LIKE ? OR phone LIKE ? OR email LIKE ?)`
		like := "%" + strings.TrimSpace(q) + "%"
		args = append(args, like, like, like)
	}
	query += ` ORDER BY id DESC LIMIT ?`
	args = append(args, limit)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Client
	for rows.Next() {
		var c Client
		var email, addr, npwp sql.NullString
		if err := rows.Scan(&c.ID, &c.OfficeID, &c.Kind, &c.FullName, &email, &c.Phone, &addr, &npwp); err != nil {
			return nil, err
		}
		if email.Valid {
			c.Email = email.String
		}
		if addr.Valid {
			c.Address = addr.String
		}
		if npwp.Valid {
			c.NPWP = npwp.String
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Get fetches one client scoped to the office (prevents cross-office access).
func (s *Store) Get(officeID, id int64) (Client, error) {
	var c Client
	var email, addr, npwp sql.NullString
	err := s.db.QueryRow(`SELECT id, office_id, kind, full_name, email, phone, address, npwp
		FROM clients WHERE id=? AND office_id=?`, id, officeID).
		Scan(&c.ID, &c.OfficeID, &c.Kind, &c.FullName, &email, &c.Phone, &addr, &npwp)
	if err != nil {
		return Client{}, err
	}
	if email.Valid {
		c.Email = email.String
	}
	if addr.Valid {
		c.Address = addr.String
	}
	if npwp.Valid {
		c.NPWP = npwp.String
	}
	return c, nil
}

// Update edits client contact data (identity fields need verification flow).
func (s *Store) Update(officeID int64, c Client) error {
	if err := validate.Required("nama", c.FullName); err != nil {
		return err
	}
	if err := validate.Phone(c.Phone); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE clients SET full_name=?, email=?, phone=?, address=?, npwp=?
		WHERE id=? AND office_id=?`, c.FullName, null(c.Email), c.Phone, null(c.Address), null(c.NPWP), c.ID, officeID)
	return err
}

func null(s string) sql.NullString {
	if strings.TrimSpace(s) == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: s, Valid: true}
}

type errString string

func errf(s string) error         { return errString(s) }
func (e errString) Error() string { return string(e) }
