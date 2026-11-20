package requests

import (
	"database/sql"
	"errors"
	"time"
)

// Valid status transitions for the service-request workflow engine.
var transitions = map[string][]string{
	"DRAFT":       {"SUBMITTED", "CANCELLED"},
	"SUBMITTED":   {"IN_REVIEW", "CANCELLED"},
	"IN_REVIEW":   {"NEED_DOCS", "APPROVED", "REJECTED"},
	"NEED_DOCS":   {"IN_REVIEW", "CANCELLED"},
	"APPROVED":    {"IN_PROGRESS"},
	"IN_PROGRESS": {"COMPLETED"},
	"COMPLETED":   {},
	"REJECTED":    {},
	"CANCELLED":   {},
}

// CanMove reports whether a status transition is legal.
func CanMove(from, to string) bool {
	for _, t := range transitions[from] {
		if t == to {
			return true
		}
	}
	return false
}

// Request is a service request (permohonan layanan).
type Request struct {
	ID         int64
	OfficeID   int64
	ClientID   int64
	ClientName string
	ServiceID  int64
	Service    string
	Status     string
	Title      string
	Notes      string
	AssigneeID *int64
	CreatedAt  time.Time
}

// Service is a catalog entry.
type Service struct {
	ID        int64
	Code      string
	Category  string
	Name      string
	BasePrice float64
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Services lists active catalog entries.
func (s *Store) Services() ([]Service, error) {
	rows, err := s.db.Query(`SELECT id, code, category, name, base_price FROM services WHERE is_active=1 ORDER BY category, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Service
	for rows.Next() {
		var sv Service
		if err := rows.Scan(&sv.ID, &sv.Code, &sv.Category, &sv.Name, &sv.BasePrice); err != nil {
			return nil, err
		}
		out = append(out, sv)
	}
	return out, rows.Err()
}

// Create opens a request in DRAFT status.
func (s *Store) Create(officeID, clientID, serviceID int64, title, notes string, createdBy *int64) (int64, error) {
	if title == "" {
		return 0, errors.New("judul permohonan wajib diisi")
	}
	res, err := s.db.Exec(`INSERT INTO service_requests(office_id, client_id, service_id, status, title, notes, created_by)
		VALUES(?,?,?,?,?,?,?)`, officeID, clientID, serviceID, "DRAFT", title, notes, createdBy)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Move transitions status with an audit event row.
func (s *Store) Move(officeID, id int64, to string, actorID *int64, note string) error {
	var from string
	if err := s.db.QueryRow(`SELECT status FROM service_requests WHERE id=? AND office_id=?`, id, officeID).Scan(&from); err != nil {
		return err
	}
	if !CanMove(from, to) {
		return errors.New("transisi status " + from + " → " + to + " tidak diizinkan")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE service_requests SET status=? WHERE id=?`, to, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO request_events(request_id, actor_id, from_status, to_status, note)
		VALUES(?,?,?,?,?)`, id, actorID, from, to, note); err != nil {
		return err
	}
	return tx.Commit()
}

// List returns requests newest first, optionally filtered by status and client.
func (s *Store) List(officeID int64, status string, clientID int64) ([]Request, error) {
	q := `SELECT r.id, r.office_id, r.client_id, c.full_name, r.service_id, sv.name, r.status, r.title, r.notes, r.assignee_id, r.created_at
		FROM service_requests r JOIN clients c ON c.id=r.client_id JOIN services sv ON sv.id=r.service_id
		WHERE r.office_id=?`
	args := []any{officeID}
	if status != "" {
		q += ` AND r.status=?`
		args = append(args, status)
	}
	if clientID != 0 {
		q += ` AND r.client_id=?`
		args = append(args, clientID)
	}
	q += ` ORDER BY r.id DESC LIMIT 200`
	rows, err := s.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Request
	for rows.Next() {
		var r Request
		var notes sql.NullString
		if err := rows.Scan(&r.ID, &r.OfficeID, &r.ClientID, &r.ClientName, &r.ServiceID,
			&r.Service, &r.Status, &r.Title, &notes, &r.AssigneeID, &r.CreatedAt); err != nil {
			return nil, err
		}
		if notes.Valid {
			r.Notes = notes.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Counts returns dashboard counters computed from real queries.
func (s *Store) Counts(officeID int64) (pending, active, completed int, err error) {
	err = s.db.QueryRow(`SELECT
		SUM(status IN ('SUBMITTED','IN_REVIEW','NEED_DOCS')),
		SUM(status IN ('APPROVED','IN_PROGRESS')),
		SUM(status='COMPLETED')
		FROM service_requests WHERE office_id=?`, officeID).Scan(&pending, &active, &completed)
	return
}
