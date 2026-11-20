package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
)

// HashPassword hashes with bcrypt (cost 12).
func HashPassword(pw string) (string, error) {
	if len(pw) < 8 {
		return "", errors.New("password must be at least 8 characters")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pw), 12)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

// CheckPassword compares password against bcrypt hash.
func CheckPassword(hash, pw string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(pw)) == nil
}

// Session is a server-side session record.
type Session struct {
	Token     string
	UserID    int64
	Role      string
	CreatedAt time.Time
	ExpiresAt time.Time
}

// Store is an in-memory session store with expiry.
// It satisfies the modular-monolith boundary; a DB-backed
// implementation can replace it without touching handlers.
type Store struct {
	mu       sync.Mutex
	ttl      time.Duration
	sessions map[string]Session
}

func NewStore(ttl time.Duration) *Store {
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &Store{ttl: ttl, sessions: map[string]Session{}}
}

func newToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Create issues a new session for userID/role.
func (s *Store) Create(userID int64, role string) (Session, error) {
	tok, err := newToken()
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	sess := Session{Token: tok, UserID: userID, Role: role, CreatedAt: now, ExpiresAt: now.Add(s.ttl)}
	s.mu.Lock()
	s.sessions[tok] = sess
	s.mu.Unlock()
	return sess, nil
}

// Get returns a session if present and unexpired.
func (s *Store) Get(token string) (Session, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok || time.Now().UTC().After(sess.ExpiresAt) {
		delete(s.sessions, token)
		return Session{}, false
	}
	return sess, true
}

// Revoke deletes a session (logout).
func (s *Store) Revoke(token string) {
	s.mu.Lock()
	delete(s.sessions, token)
	s.mu.Unlock()
}

// NewCSRFToken generates a per-form CSRF token.
func NewCSRFToken() (string, error) { return newToken() }
