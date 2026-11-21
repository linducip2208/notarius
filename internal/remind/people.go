package remind

// People + digest reminders: birthdays, expiring IDs, will annual reviews,
// and a morning digest per staff member. All in-app (+best-effort Sender).

import (
	"database/sql"
	"fmt"

	"notaryos/internal/deliver"
)

func staffOf(db *sql.DB, officeID int64) (staff []int64, err error) {
	rows, err := db.Query(`SELECT u.id FROM users u JOIN roles r ON r.id=u.role_id
		WHERE u.office_id=? AND u.is_active=1 AND r.name NOT IN ('client')`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		_ = rows.Scan(&id)
		staff = append(staff, id)
	}
	return staff, rows.Err()
}

func notifyOnce(db *sql.DB, send Sender, uid int64, title, body string) {
	var n int
	_ = db.QueryRow(`SELECT COUNT(*) FROM notifications WHERE user_id=? AND title=? AND body=?
		AND created_at > DATE_SUB(UTC_TIMESTAMP(), INTERVAL 1 DAY)`, uid, title, body).Scan(&n)
	if n > 0 {
		return
	}
	_, _ = db.Exec(`INSERT INTO notifications(user_id, title, body) VALUES(?,?,?)`, uid, title, body)
	var c deliver.Contact
	_ = db.QueryRow(`SELECT COALESCE(name,''), COALESCE(email,''), COALESCE(phone,'') FROM users WHERE id=?`, uid).Scan(&c.Name, &c.Email, &c.Phone)
	_ = send.Send(uid, c, title, body)
}

// RunPeople scans people-centric due items. All counts are new notifications.
func RunPeople(db *sql.DB, officeID int64, send Sender) (birthdays, expiringIDs, willsDue int, err error) {
	staff, err := staffOf(db, officeID)
	if err != nil {
		return 0, 0, 0, err
	}
	if len(staff) == 0 {
		return 0, 0, 0, nil
	}
	// Ulang tahun 7 hari ke depan.
	brows, _ := db.Query(`SELECT full_name, DATE_FORMAT(birth_date,'%d %b') FROM clients
		WHERE office_id=? AND birth_date IS NOT NULL AND status='ACTIVE'
		AND (DAYOFYEAR(birth_date)-DAYOFYEAR(CURDATE())+365)%365 <= 7`, officeID)
	if brows != nil {
		defer brows.Close()
		for brows.Next() {
			var name, when string
			_ = brows.Scan(&name, &when)
			for _, uid := range staff {
				notifyOnce(db, send, uid, "Ultah klien", fmt.Sprintf("%s — %s", name, when))
			}
			birthdays++
		}
	}
	// Identitas kedaluwarsa ≤30 hari.
	irows, _ := db.Query(`SELECT c.full_name, i.id_type, i.doc_expiry FROM identities i
		JOIN clients c ON c.id=i.client_id WHERE c.office_id=? AND c.status='ACTIVE'
		AND i.doc_expiry IS NOT NULL AND i.doc_expiry BETWEEN CURDATE() AND DATE_ADD(CURDATE(), INTERVAL 30 DAY)`, officeID)
	if irows != nil {
		defer irows.Close()
		for irows.Next() {
			var name, typ, exp string
			_ = irows.Scan(&name, &typ, &exp)
			for _, uid := range staff {
				notifyOnce(db, send, uid, "Identitas kedaluwarsa", fmt.Sprintf("%s — %s (%s)", name, typ, exp))
			}
			expiringIDs++
		}
	}
	// Wasiat terdaftar >1 tahun tanpa review.
	wrows, _ := db.Query(`SELECT w.id, c.full_name FROM wills w JOIN clients c ON c.id=w.client_id
		WHERE w.office_id=? AND w.status='REGISTERED' AND c.status='ACTIVE'
		AND (w.review_date IS NULL OR w.review_date < DATE_SUB(CURDATE(), INTERVAL 1 YEAR))
		AND w.created_at < DATE_SUB(NOW(), INTERVAL 1 YEAR)`, officeID)
	if wrows != nil {
		defer wrows.Close()
		for wrows.Next() {
			var id int64
			var name string
			_ = wrows.Scan(&id, &name)
			for _, uid := range staff {
				notifyOnce(db, send, uid, "Review wasiat tahunan", fmt.Sprintf("Wasiat #%d — %s", id, name))
			}
			willsDue++
		}
	}
	return birthdays, expiringIDs, willsDue, nil
}

// Digest sends one morning summary per staff member (agenda + tunggakan + antrean).
func Digest(db *sql.DB, officeID int64, send Sender) (int, error) {
	staff, err := staffOf(db, officeID)
	if err != nil || len(staff) == 0 {
		return 0, err
	}
	var appt, overdue, stuck, inq, book int
	_ = db.QueryRow(`SELECT COUNT(*) FROM appointments WHERE office_id=? AND status='SCHEDULED' AND DATE(scheduled_at)=CURDATE()`, officeID).Scan(&appt)
	_ = db.QueryRow(`SELECT COUNT(*) FROM invoices WHERE office_id=? AND status IN ('ISSUED','PARTIAL') AND due_date IS NOT NULL AND due_date < CURDATE()`, officeID).Scan(&overdue)
	_ = db.QueryRow(`SELECT COUNT(*) FROM service_requests r LEFT JOIN services s ON s.id=r.service_id WHERE r.office_id=? AND r.status IN ('SUBMITTED','IN_REVIEW','NEED_DOCS') AND r.updated_at < DATE_SUB(UTC_TIMESTAMP(), INTERVAL COALESCE(s.sla_days,3) DAY)`, officeID).Scan(&stuck)
	_ = db.QueryRow(`SELECT COUNT(*) FROM website_inquiries WHERE office_id=? AND status='BARU'`, officeID).Scan(&inq)
	_ = db.QueryRow(`SELECT COUNT(*) FROM website_bookings WHERE office_id=? AND status='MENUNGGU'`, officeID).Scan(&book)
	body := fmt.Sprintf("Hari ini: %d janji · %d invoice telat · %d permohonan macet · %d konsultasi baru · %d booking menunggu", appt, overdue, stuck, inq, book)
	for _, uid := range staff {
		notifyOnce(db, send, uid, "Ringkasan pagi", body)
	}
	return len(staff), nil
}
