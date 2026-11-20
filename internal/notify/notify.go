package notify

import "database/sql"

// Create inserts a user notification (surfaced in UI header count + portal).
func Create(db *sql.DB, userID int64, title, body string) error {
	_, err := db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?,?,?)`, userID, title, body)
	return err
}

// Unread counts unread notifications.
func Unread(db *sql.DB, userID int64) int {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND is_read=0`, userID).Scan(&n)
	return n
}

// List returns newest notifications.
func List(db *sql.DB, userID int64) ([]map[string]any, error) {
	rows, err := db.Query(`SELECT id, title, body, is_read, created_at FROM notifications WHERE user_id=? ORDER BY id DESC LIMIT 20`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]any
	for rows.Next() {
		var id int64
		var title, body string
		var read bool
		var at string
		if err := rows.Scan(&id, &title, &body, &read, &at); err != nil {
			return nil, err
		}
		out = append(out, map[string]any{"ID": id, "Title": title, "Body": body, "Read": read, "At": at})
	}
	return out, rows.Err()
}

// ReadAll marks all as read.
func ReadAll(db *sql.DB, userID int64) error {
	_, err := db.Exec(`UPDATE notifications SET is_read=1 WHERE user_id=?`, userID)
	return err
}
