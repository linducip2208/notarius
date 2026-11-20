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
	rows, err := s.db.Query(`SELECT id, client_id, deed_id, register_no, status FROM wills WHERE office_id=? ORDER BY id DESC`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Will
	for rows.Next() {
		var w Will
		var reg sql.NullString
		if err := rows.Scan(&w.ID, &w.ClientID, &w.DeedID, &reg, &w.Status); err != nil {
			return nil, err
		}
		if reg.Valid {
			w.RegisterNo = reg.String
		}
		out = append(out, w)
	}
	return out, rows.Err()
}
