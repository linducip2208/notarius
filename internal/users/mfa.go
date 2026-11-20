package users

import (
	"database/sql"

	"notaryos/internal/mfa"
)

// SetMFASecret stores a new TOTP secret (user scans QR, then confirms code).
func (s *Store) SetMFASecret(userID int64, secret string) error {
	_, err := s.db.Exec(`UPDATE users SET mfa_secret=? WHERE id=?`, secret, userID)
	return err
}

// VerifyMFA checks a TOTP code against the stored secret.
func (s *Store) VerifyMFA(userID int64, code string) bool {
	var secret sql.NullString
	if err := s.db.QueryRow(`SELECT mfa_secret FROM users WHERE id=?`, userID).Scan(&secret); err != nil {
		return false
	}
	if !secret.Valid || secret.String == "" {
		return false
	}
	return mfa.Verify(secret.String, code, now())
}

// MFAEnabled reports whether the user enrolled TOTP.
func (s *Store) MFAEnabled(userID int64) bool {
	var secret sql.NullString
	if err := s.db.QueryRow(`SELECT mfa_secret FROM users WHERE id=?`, userID).Scan(&secret); err != nil {
		return false
	}
	return secret.Valid && secret.String != ""
}

// LinkClient binds a login to a client record for portal isolation.
func (s *Store) LinkClient(userID, clientID int64) error {
	_, err := s.db.Exec(`INSERT IGNORE INTO user_client_links(user_id, client_id) VALUES(?,?)`, userID, clientID)
	return err
}

// LinkedClient returns the client record bound to a portal login (0 = none).
func (s *Store) LinkedClient(userID int64) int64 {
	var id int64
	if err := s.db.QueryRow(`SELECT client_id FROM user_client_links WHERE user_id=? LIMIT 1`, userID).Scan(&id); err != nil {
		return 0
	}
	return id
}
