package audit

import (
	"database/sql"
	"log/slog"
	"time"
)

// Entry is an immutable audit event.
type Entry struct {
	ActorID  *int64
	Action   string
	Entity   string
	EntityID string
	Detail   string
}

// Logger writes audit events to MySQL (best effort) and structured logs.
type Logger struct {
	db  *sql.DB
	log *slog.Logger
}

func New(db *sql.DB, log *slog.Logger) *Logger { return &Logger{db: db, log: log} }

// Record persists the event. DB failure never breaks the request;
// it is logged so backup monitoring can catch it.
func (l *Logger) Record(e Entry) {
	l.log.Info("audit", "action", e.Action, "entity", e.Entity, "entity_id", e.EntityID)
	if l.db == nil {
		return
	}
	_, err := l.db.Exec(`INSERT INTO audit_logs(actor_id, action, entity, entity_id, detail, created_at)
		VALUES(?,?,?,?,?,?)`, e.ActorID, e.Action, e.Entity, e.EntityID, e.Detail, time.Now().UTC())
	if err != nil {
		l.log.Error("audit persist failed", "err", err)
	}
}
