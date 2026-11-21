// Package remind generates operational reminders as in-app notifications:
// H-1 appointments, overdue invoices, tasks due. Delivery to real gateways
// (e.g. WhatsApp) is a Sender adapter; default is in-app notify rows.
package remind

import (
	"database/sql"
	"fmt"
	"time"
)

// Sender delivers a reminder outside the app (WA gateway, email, ...).
// LogSender is the honest default: tercatat, tidak diklaim terkirim ke WA.
type Sender interface {
	Send(to, message string) error
}

type LogSender struct{}

func (LogSender) Send(to, message string) error { return nil }

// RunOnce scans one office and creates notifications. Returns counts.
// SLA: permohonan SUBMITTED/IN_REVIEW/NEEDS_DOCS yang tak bergerak >3 hari
// dieskalasi ke assignee (atau seluruh staf bila tanpa assignee).
func RunOnce(db *sql.DB, officeID int64, send Sender) (appt, overdue, tasks, sla int, err error) {
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	today := time.Now().Format("2006-01-02")

	var users []struct {
		id   int64
		role string
	}
	rows, err := db.Query(`SELECT u.id, r.name FROM users u JOIN roles r ON r.id=u.role_id
		WHERE u.office_id=? AND u.is_active=1`, officeID)
	if err != nil {
		return 0, 0, 0, 0, err
	}
	for rows.Next() {
		var u struct {
			id   int64
			role string
		}
		_ = rows.Scan(&u.id, &u.role)
		users = append(users, u)
	}
	rows.Close()
	staff, finance := []int64{}, []int64{}
	for _, u := range users {
		if u.role == "client" {
			continue
		}
		staff = append(staff, u.id)
		if u.role == "finance" || u.role == "office_admin" || u.role == "super_admin" {
			finance = append(finance, u.id)
		}
	}
	notify := func(uid int64, title, body string) {
		var n int
		_ = db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND title=? AND body=?
			AND created_at > DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 DAY)`, uid, title, body).Scan(&n)
		if n > 0 {
			return // sudah diingatkan 24 jam terakhir — anti-spam
		}
		_, _ = db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?,?,?)`, uid, title, body)
	}
	// H-1 appointments.
	arows, _ := db.Query(`SELECT c.full_name, a.purpose, a.scheduled_at FROM appointments a
		JOIN clients c ON c.id=a.client_id
		WHERE a.office_id=? AND a.status='SCHEDULED' AND DATE(a.scheduled_at)=?`, officeID, tomorrow)
	if arows != nil {
		defer arows.Close()
		for arows.Next() {
			var name, purpose, at string
			_ = arows.Scan(&name, &purpose, &at)
			msg := fmt.Sprintf("Besok %s — %s (%s)", at[11:16], name, purpose)
			for _, uid := range staff {
				notify(uid, "Penghadap besok", msg)
			}
			appt++
		}
	}
	// Overdue invoices.
	irows, _ := db.Query(`SELECT number, due_date FROM invoices WHERE office_id=?
		AND status IN ('ISSUED','PARTIAL') AND due_date IS NOT NULL AND due_date < CURDATE()`, officeID)
	if irows != nil {
		defer irows.Close()
		for irows.Next() {
			var num, due string
			_ = irows.Scan(&num, &due)
			for _, uid := range finance {
				notify(uid, "Invoice jatuh tempo", fmt.Sprintf("%s jatuh tempo %s", num, due))
			}
			overdue++
		}
	}
	// Tasks due today/tomorrow.
	trows, _ := db.Query(`SELECT id, title, created_by FROM tasks WHERE office_id=?
		AND status='OPEN' AND due_date IS NOT NULL AND due_date <= DATE_ADD(CURDATE(), INTERVAL 1 DAY)`, officeID)
	if trows != nil {
		defer trows.Close()
		for trows.Next() {
			var id int64
			var title string
			var by sql.NullInt64
			_ = trows.Scan(&id, &title, &by)
			msg := fmt.Sprintf("Tugas #%d: %s", id, title)
			if by.Valid {
				notify(by.Int64, "Tugas mendesak", msg)
			} else {
				for _, uid := range staff {
					notify(uid, "Tugas mendesak", msg)
				}
			}
			_ = today
			tasks++
		}
	}
	// SLA eskalasi: permohonan macet >3 hari tanpa pergerakan.
	srows, _ := db.Query(`SELECT id, title, status, assignee_id, DATEDIFF(UTC_TIMESTAMP(), updated_at)
		FROM service_requests WHERE office_id=? AND status IN ('SUBMITTED','IN_REVIEW','NEEDS_DOCS')
		AND updated_at < DATE_SUB(UTC_TIMESTAMP(), INTERVAL 3 DAY)`, officeID)
	if srows != nil {
		defer srows.Close()
		for srows.Next() {
			var id int64
			var title, status string
			var assignee sql.NullInt64
			var age int
			_ = srows.Scan(&id, &title, &status, &assignee, &age)
			msg := fmt.Sprintf("#%d %s — %s, macet %d hari", id, title, status, age)
			if assignee.Valid {
				notify(assignee.Int64, "Permohonan macet (SLA)", msg)
			} else {
				for _, uid := range staff {
					notify(uid, "Permohonan macet (SLA)", msg)
				}
			}
			sla++
		}
	}
	return appt, overdue, tasks, sla, nil
}
