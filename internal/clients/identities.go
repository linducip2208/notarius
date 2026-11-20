package clients

import (
	"database/sql"
	"time"

	"notaryos/internal/cryptoxx"
	"notaryos/internal/validate"
)

// Identity is one identity document row. NIK plaintext is NEVER stored;
// nik_cipher holds AES-256-GCM ciphertext, nik_last4 supports lookup.
type Identity struct {
	ID         int64
	ClientID   int64
	IDType     string
	NIKLast4   string
	FullName   string
	Verified   bool
	VerifiedBy *int64
	CreatedAt  time.Time
}

// AddIdentity stores an identity; NIK is encrypted when a field key is configured.
func (s *Store) AddIdentity(clientID int64, idType, nik, fullName string) (int64, error) {
	switch idType {
	case "KTP", "KK", "NPWP", "PASSPORT", "OTHER":
	default:
		return 0, errf("jenis identitas tidak dikenal")
	}
	if idType == "KTP" {
		if err := validate.NIK(nik); err != nil {
			return 0, err
		}
	}
	if err := validate.Required("nama pada identitas", fullName); err != nil {
		return 0, err
	}
	var cipher sql.NullString
	var last4 sql.NullString
	if nik != "" {
		if len(nik) >= 4 {
			last4 = sql.NullString{String: nik[len(nik)-4:], Valid: true}
		}
		if key, err := cryptoxx.Key(); err == nil {
			sealed, err := cryptoxx.Encrypt(key, []byte(nik))
			if err != nil {
				return 0, err
			}
			cipher = sql.NullString{String: sealed, Valid: true}
		}
		// Without a field key we store only last4 (never plaintext) — see TD-02.
	}
	res, err := s.db.Exec(`INSERT INTO identities(client_id, id_type, nik_cipher, nik_last4, full_name)
		VALUES(?,?,?,?,?)`, clientID, idType, cipher, last4, fullName)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// Identities lists a client's identity documents (no NIK plaintext leaves the DB).
func (s *Store) Identities(clientID int64) ([]Identity, error) {
	rows, err := s.db.Query(`SELECT id, client_id, id_type, nik_last4, full_name, verified, verified_by, created_at
		FROM identities WHERE client_id=? ORDER BY id`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Identity
	for rows.Next() {
		var it Identity
		var last4 sql.NullString
		if err := rows.Scan(&it.ID, &it.ClientID, &it.IDType, &last4, &it.FullName, &it.Verified, &it.VerifiedBy, &it.CreatedAt); err != nil {
			return nil, err
		}
		if last4.Valid {
			it.NIKLast4 = last4.String
		}
		out = append(out, it)
	}
	return out, rows.Err()
}

// VerifyIdentity marks human verification (staff confirms OCR-extracted data).
func (s *Store) VerifyIdentity(id, verifierID int64) error {
	_, err := s.db.Exec(`UPDATE identities SET verified=1, verified_by=?, verified_at=UTC_TIMESTAMP() WHERE id=?`,
		verifierID, id)
	return err
}

// Rep is a client representative.
type Rep struct {
	ID        int64
	Name      string
	Position  string
	Phone     string
	Authority string
}

// AddRep stores a representative.
func (s *Store) AddRep(clientID int64, r Rep) error {
	if err := validate.Required("nama perwakilan", r.Name); err != nil {
		return err
	}
	_, err := s.db.Exec(`INSERT INTO client_representatives(client_id, name, position, phone, authority_note)
		VALUES(?,?,?,?,?)`, clientID, r.Name, null(r.Position), null(r.Phone), null(r.Authority))
	return err
}

// Reps lists representatives.
func (s *Store) Reps(clientID int64) ([]Rep, error) {
	rows, err := s.db.Query(`SELECT id, name, position, phone, authority_note FROM client_representatives WHERE client_id=?`, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Rep
	for rows.Next() {
		var r Rep
		var pos, phone, auth sql.NullString
		if err := rows.Scan(&r.ID, &r.Name, &pos, &phone, &auth); err != nil {
			return nil, err
		}
		if pos.Valid {
			r.Position = pos.String
		}
		if phone.Valid {
			r.Phone = phone.String
		}
		if auth.Valid {
			r.Authority = auth.String
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
