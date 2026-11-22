package deeds

import (
	"database/sql"
	"errors"
)

// Will is a will-registration record. Access is restricted to notaries.
type Will struct {
	ID         int64
	ClientID   int64
	DeedID     *int64
	RegisterNo string
	Status     string
	ReviewDate string
	OpenedBy   int64
	Witness    string
	OpenedAt   string
}

type WillStore struct{ db *sql.DB }

func NewWillStore(db *sql.DB) *WillStore { return &WillStore{db: db} }

// Register records a will. Only notary roles may call (enforced in handler).
func (s *WillStore) Register(officeID, clientID int64, deedID *int64, registerNo string) (int64, error) {
	if registerNo == "" {
		return 0, errors.New("nomor register wasiat wajib diisi")
	}
	res, err := s.db.Exec(`INSERT INTO wills(office_id, client_id, deed_id, register_no, status, restricted)
		VALUES(?,?,?,?,'REGISTERED',1)`, officeID, clientID, deedID, registerNo)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// List returns wills for the office (caller must be authorized).
func (s *WillStore) List(officeID int64) ([]Will, error) {
	rows, err := s.db.Query(`SELECT id, client_id, deed_id, register_no, status, review_date,
		COALESCE(opened_by,0), COALESCE(opened_witness,''), COALESCE(opened_at,'') FROM wills WHERE office_id=? ORDER BY id DESC`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Will
	for rows.Next() {
		var w Will
		var reg, rev sql.NullString
		if err := rows.Scan(&w.ID, &w.ClientID, &w.DeedID, &reg, &w.Status, &rev, &w.OpenedBy, &w.Witness, &w.OpenedAt); err != nil {
			return nil, err
		}
		if reg.Valid {
			w.RegisterNo = reg.String
		}
		w.ReviewDate = rev.String
		out = append(out, w)
	}
	return out, rows.Err()
}

// Handover records serah-terima minuta wasiat (F-014: hanya notaris, tercatat + audit).
func (s *WillStore) Handover(officeID, willID int64, handedTo, note string) error {
	if handedTo == "" {
		return errors.New("penerima serah-terima wajib diisi")
	}
	res, err := s.db.Exec(`UPDATE wills SET status='ARCHIVED', handed_over=1, handed_to=?, handed_at=UTC_TIMESTAMP(), handover_note=?
		WHERE id=? AND office_id=? AND handed_over=0`, handedTo, nullStr(note), willID, officeID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return errors.New("wasiat tidak ditemukan atau sudah diserahterimakan")
	}
	return nil
}

func nullStr(s string) any {
	if s == "" {
		return nil
	}
	return s
}
