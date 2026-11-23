package finance

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"notaryos/internal/money"
)

// Invoice aggregates lines; all math uses int64 rupiah.
type Invoice struct {
	ID          int64
	OfficeID    int64
	ClientID    int64
	Number      string
	Subtotal    int64
	Tax         int64
	Total       int64
	Paid        int64
	Status      string
	DueDate     string
	Outstanding int64
	// QuotationID terisi bila invoice lahir dari penawaran (migrasi 030,
	// NULLABLE — invoice manual existing tidak tersentuh).
	QuotationID *int64
}

// Line is one invoice line.
type Line struct {
	Description string
	Qty         int
	UnitPrice   int64
}

type Store struct{ db *sql.DB }

func NewStore(db *sql.DB) *Store { return &Store{db: db} }

// deci converts a DECIMAL(18,2) string ("5550000.00") to whole rupiah.
// The MySQL driver returns DECIMAL as bytes; scanning into int64 fails.
func deci(s string) int64 {
	// DECIMAL(18,2) arrives as "5550000.00"; keep the whole-rupiah part.
	s = strings.TrimSpace(s)
	neg := false
	if strings.HasPrefix(s, "-") {
		neg = true
		s = s[1:]
	}
	if i := strings.IndexByte(s, '.'); i >= 0 {
		s = s[:i]
	}
	var n int64
	for _, r := range s {
		if r >= '0' && r <= '9' {
			n = n*10 + int64(r-'0')
		}
	}
	if neg {
		n = -n
	}
	return n
}

// CreateInvoice computes subtotal/tax/total in Go (decimal-safe) then persists.
func (s *Store) CreateInvoice(officeID int64, clientID int64, requestID *int64, number string, lines []Line, taxBps int, due string) (int64, error) {
	if len(lines) == 0 {
		return 0, errors.New("invoice membutuhkan minimal satu rincian")
	}
	var sub int64
	for _, l := range lines {
		if l.Qty <= 0 || l.UnitPrice < 0 {
			return 0, errors.New("qty/harga baris tidak valid")
		}
		lt, err := money.Mul(l.UnitPrice, l.Qty)
		if err != nil {
			return 0, err
		}
		sub = money.Add(sub, lt)
	}
	tax := money.PPN(sub, taxBps) - sub
	total := money.Add(sub, tax)
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	var dueVal any
	if due != "" {
		dueVal = due
	}
	res, err := tx.Exec(`INSERT INTO invoices(office_id, client_id, request_id, number, subtotal, tax, total, status, due_date)
		VALUES(?,?,?,?,?,?,?,'ISSUED',?)`, officeID, clientID, requestID, number, sub, tax, total, dueVal)
	if err != nil {
		if isDupErr(err) {
			return 0, errors.New("nomor invoice sudah dipakai")
		}
		return 0, err
	}
	id, _ := res.LastInsertId()
	for _, l := range lines {
		lt, _ := money.Mul(l.UnitPrice, l.Qty)
		if _, err := tx.Exec(`INSERT INTO invoice_lines(invoice_id, description, qty, unit_price, line_total)
			VALUES(?,?,?,?,?)`, id, l.Description, l.Qty, l.UnitPrice, lt); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// VoidInvoice cancels an unpaid invoice with reason (never deletes history).
func (s *Store) VoidInvoice(officeID, invoiceID int64, reason string) error {
	if strings.TrimSpace(reason) == "" {
		return errors.New("alasan void wajib diisi")
	}
	res, err := s.db.Exec(`UPDATE invoices SET status='VOID' WHERE id=? AND office_id=? AND status IN ('ISSUED','PARTIAL')`, invoiceID, officeID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("hanya invoice ISSUED/PARTIAL yang dapat di-void")
	}
	return nil
}

func isDupErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "1062") || strings.Contains(msg, "Duplicate entry")
}

// RecordPayment applies a payment and updates invoice status (office-scoped).
func (s *Store) RecordPayment(officeID, invoiceID, amount int64, method, ref string, receivedBy *int64) error {
	if amount <= 0 {
		return errors.New("nominal pembayaran harus positif")
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var stotal, spaid string
	var status string
	if err := tx.QueryRow(`SELECT total, paid, status FROM invoices WHERE id=? AND office_id=? FOR UPDATE`, invoiceID, officeID).
		Scan(&stotal, &spaid, &status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("invoice tidak ditemukan")
		}
		return err
	}
	total, paid := deci(stotal), deci(spaid)
	if status == "VOID" {
		return errors.New("invoice void tidak dapat dibayar")
	}
	if money.Add(paid, amount) > total {
		return fmt.Errorf("kelebihan bayar: sisa %s", money.FormatIDR(money.Sub(total, paid)))
	}
	if _, err := tx.Exec(`INSERT INTO payments(invoice_id, amount, method, reference, received_by)
		VALUES(?,?,?,?,?)`, invoiceID, amount, method, ref, receivedBy); err != nil {
		return err
	}
	newPaid := money.Add(paid, amount)
	newStatus := "PARTIAL"
	if newPaid == total {
		newStatus = "PAID"
	}
	res, err := tx.Exec(`UPDATE invoices SET paid=?, status=? WHERE id=? AND office_id=?`, newPaid, newStatus, invoiceID, officeID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("invoice tidak ditemukan")
	}
	return tx.Commit()
}

// Outstanding sums unpaid balances for the office (real query, no fake stats).
func (s *Store) Outstanding(officeID int64) (int64, error) {
	var v sql.NullString
	if err := s.db.QueryRow(`SELECT SUM(total-paid) FROM invoices WHERE office_id=? AND status IN ('ISSUED','PARTIAL')`,
		officeID).Scan(&v); err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return deci(v.String), nil
}

// Revenue sums payments received in a month (YYYY-MM).
func (s *Store) Revenue(officeID int64, month string) (int64, error) {
	if _, err := time.Parse("2006-01", month); err != nil {
		return 0, errors.New("format bulan harus YYYY-MM")
	}
	var v sql.NullString
	err := s.db.QueryRow(`SELECT SUM(p.amount) FROM payments p JOIN invoices i ON i.id=p.invoice_id
		WHERE i.office_id=? AND DATE_FORMAT(p.paid_at,'%Y-%m')=?`, officeID, month).Scan(&v)
	if err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return deci(v.String), nil
}

// List lists invoices newest first.
func (s *Store) List(officeID int64) ([]Invoice, error) {
	rows, err := s.db.Query(`SELECT id, office_id, client_id, number, subtotal, tax, total, paid, status, due_date, quotation_id
		FROM invoices WHERE office_id=? ORDER BY id DESC LIMIT 200`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Invoice
	for rows.Next() {
		var iv Invoice
		var due sql.NullString
		var qid sql.NullInt64
		var ssub, stax, stotal, spaid string
		if err := rows.Scan(&iv.ID, &iv.OfficeID, &iv.ClientID, &iv.Number, &ssub,
			&stax, &stotal, &spaid, &iv.Status, &due, &qid); err != nil {
			return nil, err
		}
		iv.Subtotal, iv.Tax, iv.Total, iv.Paid = deci(ssub), deci(stax), deci(stotal), deci(spaid)
		if due.Valid {
			iv.DueDate = due.String
		}
		if qid.Valid {
			iv.QuotationID = &qid.Int64
		}
		iv.Outstanding = money.Sub(iv.Total, iv.Paid)
		out = append(out, iv)
	}
	return out, rows.Err()
}
