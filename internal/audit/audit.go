package audit

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"time"
)

// chainKey signs the hash chain (server-side secret, never logged).
func chainKey() []byte {
	if k := os.Getenv("NOTARYOS_CHAIN_KEY"); k != "" {
		return []byte(k)
	}
	if k := os.Getenv("NOTARYOS_FIELD_KEY"); k != "" {
		return []byte("chain:" + k)
	}
	return []byte("chain:default-dev-only")
}

// Entry is an immutable audit event.
type Entry struct {
	ActorID  *int64
	Action   string
	Entity   string
	EntityID string
	Detail   string
	// OfficeID is optional: owning office of the event. Pointer (not plain
	// int64) so Record's signature stays unchanged; nil writes NULL =
	// legacy/system rows whose office is unknown.
	OfficeID *int64
}

// Logger writes audit events to MySQL (best effort) and structured logs.
type Logger struct {
	db  *sql.DB
	log *slog.Logger
}

func New(db *sql.DB, log *slog.Logger) *Logger { return &Logger{db: db, log: log} }

// Record persists the event with a hash chain (prev_hash + HMAC chain_hash).
// DB failure never breaks the request; it is logged so backup monitoring can catch it.
// Chain is best-effort serialized via tx; concurrent writers serialize on the tail row.
func (l *Logger) Record(e Entry) {
	l.log.Info("audit", "action", e.Action, "entity", e.Entity, "entity_id", e.EntityID)
	if l.db == nil {
		return
	}
	tx, err := l.db.Begin()
	if err != nil {
		l.log.Error("audit persist failed", "err", err)
		return
	}
	defer tx.Rollback()
	var lastID int64
	var lastChain sql.NullString
	_ = tx.QueryRow(`SELECT id, chain_hash FROM audit_logs ORDER BY id DESC LIMIT 1 FOR UPDATE`).Scan(&lastID, &lastChain)
	prev := ""
	if lastChain.Valid {
		prev = lastChain.String
	}
	now := time.Now().UTC().Truncate(time.Second) // samakan presisi TIMESTAMP MySQL (detik)
	var actor string
	if e.ActorID != nil {
		actor = strconv.FormatInt(*e.ActorID, 10)
	}
	payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", lastID, actor, e.Action, e.Entity, e.EntityID, e.Detail, now.Format(time.RFC3339Nano))
	mac := hmac.New(sha256.New, chainKey())
	_, _ = mac.Write([]byte(prev + "|" + payload))
	chain := fmt.Sprintf("%x", mac.Sum(nil))
	if _, err := tx.Exec(`INSERT INTO audit_logs(actor_id, action, entity, entity_id, detail, office_id, created_at, prev_hash, chain_hash)
		VALUES(?,?,?,?,?,?,?,?,?)`, e.ActorID, e.Action, e.Entity, e.EntityID, e.Detail, e.OfficeID, now, prev, chain); err != nil {
		l.log.Error("audit persist failed", "err", err)
		return
	}
	if err := tx.Commit(); err != nil {
		l.log.Error("audit persist failed", "err", err)
	}
}

// Verify checks the hash chain over the last `limit` rows (0 = all, cap 10000).
func (l *Logger) Verify(limit int) error {
	if l.db == nil {
		return fmt.Errorf("no db")
	}
	q := `SELECT id, actor_id, action, entity, entity_id, detail, created_at, prev_hash, chain_hash FROM audit_logs ORDER BY id DESC`
	if limit > 0 {
		if limit > 10000 {
			limit = 10000
		}
		q += fmt.Sprintf(" LIMIT %d", limit)
	}
	rows, err := l.db.Query(q)
	if err != nil {
		return err
	}
	defer rows.Close()
	type rec struct {
		id             int64
		actor          sql.NullInt64
		action, entity string
		entityID       sql.NullString
		detail         sql.NullString
		at             time.Time
		prev, chain    sql.NullString
	}
	var list []rec
	for rows.Next() {
		var r rec
		if err := rows.Scan(&r.id, &r.actor, &r.action, &r.entity, &r.entityID, &r.detail, &r.at, &r.prev, &r.chain); err != nil {
			return err
		}
		list = append(list, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	// Walk oldest-first recomputing the chain. Rows without chain_hash predate
	// the chain feature (legacy) — skipped, never treated as verified.
	var chained []rec
	for i := len(list) - 1; i >= 0; i-- {
		if list[i].chain.String == "" {
			continue
		}
		chained = append(chained, list[i])
	}
	var prevID int64
	prevChain := ""
	for _, r := range chained {
		if r.prev.String != prevChain {
			return fmt.Errorf("rantai putus pada id %d", r.id)
		}
		actor := ""
		if r.actor.Valid {
			actor = strconv.FormatInt(r.actor.Int64, 10)
		}
		payload := fmt.Sprintf("%d|%s|%s|%s|%s|%s|%s", prevID, actor, r.action, r.entity, r.entityID.String, r.detail.String, r.at.UTC().Format(time.RFC3339Nano))
		mac := hmac.New(sha256.New, chainKey())
		_, _ = mac.Write([]byte(prevChain + "|" + payload))
		if got := fmt.Sprintf("%x", mac.Sum(nil)); got != r.chain.String {
			return fmt.Errorf("hash tidak cocok pada id %d", r.id)
		}
		prevID, prevChain = r.id, r.chain.String
	}
	return nil
}
