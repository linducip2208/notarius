package requests

import (
	"database/sql"
	"errors"
	"time"
)

// Detail is a request with its event timeline.
type Detail struct {
	Request
	Events []Event
}

// Event is one status-change audit row.
type Event struct {
	From, To, Note string
	Actor          string
	At             time.Time
}

// Get loads a request with its timeline, office-scoped.
func (s *Store) Get(officeID, id int64) (Detail, error) {
	var d Detail
	var notes sql.NullString
	err := s.db.QueryRow(`SELECT r.id, r.office_id, r.client_id, c.full_name, r.service_id, sv.name,
		r.status, r.title, r.notes, r.assignee_id, r.created_at
		FROM service_requests r JOIN clients c ON c.id=r.client_id JOIN services sv ON sv.id=r.service_id
		WHERE r.id=? AND r.office_id=?`, id, officeID).
		Scan(&d.ID, &d.OfficeID, &d.ClientID, &d.ClientName, &d.ServiceID, &d.Service,
			&d.Status, &d.Title, &notes, &d.AssigneeID, &d.CreatedAt)
	if err != nil {
		return Detail{}, err
	}
	if notes.Valid {
		d.Notes = notes.String
	}
	rows, err := s.db.Query(`SELECT e.from_status, e.to_status, e.note, u.name, e.created_at
		FROM request_events e LEFT JOIN users u ON u.id=e.actor_id
		WHERE e.request_id=? ORDER BY e.id`, id)
	if err != nil {
		return Detail{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var e Event
		var from, note, actor sql.NullString
		if err := rows.Scan(&from, &e.To, &note, &actor, &e.At); err != nil {
			return Detail{}, err
		}
		if from.Valid {
			e.From = from.String
		}
		if note.Valid {
			e.Note = note.String
		}
		if actor.Valid {
			e.Actor = actor.String
		}
		d.Events = append(d.Events, e)
	}
	return d, rows.Err()
}

// Appointment is an office visit.
type Appointment struct {
	ID        int64
	ClientID  int64
	RequestID *int64
	At        time.Time
	Purpose   string
	Status    string
}

// Appointments lists upcoming appointments for the office.
func (s *Store) Appointments(officeID int64) ([]Appointment, error) {
	rows, err := s.db.Query(`SELECT id, client_id, request_id, scheduled_at, purpose, status
		FROM appointments WHERE office_id=? ORDER BY scheduled_at LIMIT 100`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Appointment
	for rows.Next() {
		var a Appointment
		if err := rows.Scan(&a.ID, &a.ClientID, &a.RequestID, &a.At, &a.Purpose, &a.Status); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// Schedule creates an appointment.
func (s *Store) Schedule(officeID, clientID int64, requestID *int64, at time.Time, purpose string) (int64, error) {
	if purpose == "" {
		return 0, errors.New("keperluan kunjungan wajib diisi")
	}
	if at.Before(time.Now().Add(-time.Hour)) {
		return 0, errors.New("jadwal tidak boleh di masa lalu")
	}
	// Simpan waktu dinding lokal (WIB) apa adanya: dashboard & pengingat
	// membandingkan DATE()/CURDATE() dalam zona yang sama. Jangan .UTC().
	res, err := s.db.Exec(`INSERT INTO appointments(office_id, client_id, request_id, scheduled_at, purpose)
		VALUES(?,?,?,?,?)`, officeID, clientID, requestID, at.Format("2006-01-02 15:04:05"), purpose)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// SetAppointmentStatus moves SCHEDULED → DONE/CANCELLED/NOSHOW (office-scoped).
// Backward moves out of terminal states (e.g. DONE → SCHEDULED) are rejected.
func (s *Store) SetAppointmentStatus(officeID, id int64, to string) error {
	switch to {
	case "DONE", "CANCELLED", "NOSHOW", "SCHEDULED":
	default:
		return errors.New("status janji tidak dikenal")
	}
	var cur string
	if err := s.db.QueryRow(`SELECT status FROM appointments WHERE id=? AND office_id=?`, id, officeID).Scan(&cur); err != nil {
		return errors.New("janji tidak ditemukan")
	}
	if cur != "SCHEDULED" && to == "SCHEDULED" {
		return errors.New("janji yang sudah selesai/batal tidak dapat dijadwalkan ulang — buat janji baru")
	}
	res, err := s.db.Exec(`UPDATE appointments SET status=? WHERE id=? AND office_id=?`, to, id, officeID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("janji tidak ditemukan")
	}
	return nil
}
