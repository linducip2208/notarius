package gov

import (
	"database/sql"
	"encoding/json"
	"errors"
)

// Store persists assisted submissions.
type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// Create drafts an assisted submission with its standard checklist.
func (s *Store) Create(officeID int64, ch Channel, ref string) (int64, error) {
	sub := NewSubmission(ch, ref)
	raw, _ := json.Marshal(sub.Checklist)
	res, err := s.db.Exec(`INSERT INTO gov_submissions(office_id, channel, reference, status, checklist)
		VALUES(?,?,?,?,?)`, officeID, string(ch), ref, string(StatusDraft), string(raw))
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Advance moves status forward with optional receipt evidence.
func (s *Store) Advance(id int64, to Status, receiptNo, receiptRef, notes string) error {
	return s.AdvanceScoped(0, id, to, receiptNo, receiptRef, notes)
}

// AdvanceScoped is Advance with office isolation (0 = skip check, legacy).
func (s *Store) AdvanceScoped(officeID, id int64, to Status, receiptNo, receiptRef, notes string) error {
	var cur string
	var oid int64
	if err := s.db.QueryRow(`SELECT status, office_id FROM gov_submissions WHERE id=?`, id).Scan(&cur, &oid); err != nil {
		return err
	}
	if officeID != 0 && oid != officeID {
		return errors.New("pengajuan tidak ditemukan")
	}
	tmp := Submission{Status: Status(cur)}
	if err := tmp.Advance(to); err != nil {
		return err
	}
	_, err := s.db.Exec(`UPDATE gov_submissions SET status=?, receipt_no=?, receipt_ref=?, notes=? WHERE id=?`,
		string(to), receiptNo, receiptRef, notes, id)
	return err
}

// ToggleCheck flips one checklist item and persists the JSON.
func (s *Store) ToggleCheck(id int64, index int) error {
	return s.ToggleCheckScoped(0, id, index)
}

// ToggleCheckScoped is ToggleCheck with office isolation.
func (s *Store) ToggleCheckScoped(officeID, id int64, index int) error {
	var raw sql.NullString
	var oid int64
	if err := s.db.QueryRow(`SELECT checklist, office_id FROM gov_submissions WHERE id=?`, id).Scan(&raw, &oid); err != nil {
		return err
	}
	if officeID != 0 && oid != officeID {
		return errors.New("pengajuan tidak ditemukan")
	}
	var items []CheckItem
	if raw.Valid && raw.String != "" {
		_ = json.Unmarshal([]byte(raw.String), &items)
	}
	if index < 0 || index >= len(items) {
		return errors.New("item checklist tidak ditemukan")
	}
	items[index].Done = !items[index].Done
	enc, _ := json.Marshal(items)
	_, err := s.db.Exec(`UPDATE gov_submissions SET checklist=? WHERE id=?`, string(enc), id)
	return err
}

// List returns submissions for the office.
func (s *Store) List(officeID int64) ([]Submission, error) {
	rows, err := s.db.Query(`SELECT id, channel, reference, status, checklist, receipt_no, notes, updated_at, follow_up
		FROM gov_submissions WHERE office_id=? ORDER BY id DESC LIMIT 200`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Submission
	for rows.Next() {
		var x Submission
		var ch, st, cl sql.NullString
		var rcpt, notes, fu sql.NullString
		if err := rows.Scan(&x.ID, &ch, &x.Reference, &st, &cl, &rcpt, &notes, &x.UpdatedAt, &fu); err != nil {
			return nil, err
		}
		if ch.Valid {
			x.Channel = Channel(ch.String)
		}
		if st.Valid {
			x.Status = Status(st.String)
		}
		if cl.Valid && cl.String != "" {
			_ = json.Unmarshal([]byte(cl.String), &x.Checklist)
		}
		if rcpt.Valid {
			x.ReceiptNo = rcpt.String
		}
		if notes.Valid {
			x.Notes = notes.String
		}
		x.FollowUp = fu.String
		out = append(out, x)
	}
	return out, rows.Err()
}
