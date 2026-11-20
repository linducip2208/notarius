package auth

import (
	"database/sql"
	"time"
)

// SessionStore abstracts session persistence. The in-memory Store and the
// SQLStore below are interchangeable; handlers depend only on this interface.
type SessionStore interface {
	Create(userID int64, role string) (Session, error)
	Get(token string) (Session, bool)
	Revoke(token string)
}

// SQLStore persists sessions in MySQL so logins survive restarts and
// work behind multiple app instances (resolves TD-01).
type SQLStore struct {
	db  *sql.DB
	ttl time.Duration
}

func NewSQLStore(db *sql.DB, ttl time.Duration) *SQLStore {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &SQLStore{db: db, ttl: ttl}
}

func (s *SQLStore) Create(userID int64, role string) (Session, error) {
	tok, err := newToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	sess := Session{Token: tok, UserID: userID, Role: role, CreatedAt: now, ExpiresAt: now.Add(s.ttl)}
	_, err = s.db.Exec(`INSERT INTO sessions(token, user_id, role, expires_at) VALUES(?,?,?,?)`,
		tok, userID, role, sess.ExpiresAt)
	if err != nil {
		return Session{}, err
	}
	return sess, nil
}

func (s *SQLStore) Get(token string) (Session, bool) {
	var sess Session
	var uid int64
	err := s.db.QueryRow(`SELECT token, user_id, role, expires_at FROM sessions WHERE token=?`, token).
		Scan(&sess.Token, &uid, &sess.Role, &sess.ExpiresAt)
	if err != nil {
		return Session{}, false
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_, _ = s.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
		return Session{}, false
	}
	sess.UserID = uid
	return sess, true
}

func (s *SQLStore) Revoke(token string) {
	_, _ = s.db.Exec(`DELETE FROM sessions WHERE token=?`, token)
}

// Cleanup removes expired rows; call periodically (e.g. daily cron/worker).
func (s *SQLStore) Cleanup() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM sessions WHERE expires_at < UTC_TIMESTAMP()`)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}
