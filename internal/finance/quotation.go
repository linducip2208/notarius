package finance

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"notaryos/internal/money"
)

// Quotation adalah dokumen penawaran resmi terikat permohonan.
// BUKAN janji tarif final: angka contoh dari tarif layanan, dinegosiasi manual.
type Quotation struct {
	ID         int64
	OfficeID   int64
	RequestID  *int64
	Number     string
	ValidUntil string
	Status     string
	Notes      string
	Total      int64
	Lines      []QuotationLine
}

// QuotationLine adalah satu baris rincian penawaran.
type QuotationLine struct {
	ID          int64
	QuotationID int64
	Description string
	Qty         int
	UnitPrice   int64
	LineTotal   int64
}

// quotTransitions adalah alur status yang jujur: ACCEPTED hanya lahir dari
// ConvertToInvoice (ada invoice nyata), tidak bisa di-set manual agar invarian
// "ACCEPTED => ada invoice" selalu benar.
var quotTransitions = map[string][]string{
	"DRAFT": {"SENT", "REJECTED", "EXPIRED"},
	"SENT":  {"ACCEPTED", "REJECTED", "EXPIRED"},
}

// IsExpired menilai apakah penawaran lewat tanggal berlaku.
// Format: YYYY-MM-DD untuk validUntil & today; kosong/tak valid = false.
// Batas hari-H belum kedaluwarsa (validUntil < today), selaras dengan
// klausul SQL `valid_until < CURDATE()` di ExpireOverdueQuotations.
func IsExpired(validUntil, today string) bool {
	vu, err := time.Parse("2006-01-02", strings.TrimSpace(validUntil))
	if err != nil {
		return false
	}
	t, err := time.Parse("2006-01-02", strings.TrimSpace(today))
	if err != nil {
		return false
	}
	return vu.Before(t)
}

// nextQuotationNumber mengalokasi nomor Q-{office}-{YYYY}-{NNN} via MAX+1.
// JUJUR: tanpa klaim anti-race sempurna — tidak ada lock gap untuk suffix
// berikutnya (kunci baris kantor seperti repertorium tidak dipakai agar
// pembuatan penawaran tetap ringan). Jaring pengaman: UNIQUE
// uq_quot_office_number + retry 2x di CreateQuotation (tiru pola deeds.Approve).
func (s *Store) nextQuotationNumber(officeID int64) (string, error) {
	prefix := fmt.Sprintf("Q-%d-%d-", officeID, time.Now().Year())
	rows, err := s.db.Query(`SELECT number FROM quotations WHERE office_id=? AND number LIKE ?`, officeID, prefix+"%")
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var max int64
	// Suffix = segmen numerik setelah "-" terakhir; nomor non-standar diabaikan.
	for rows.Next() {
		var num string
		if err := rows.Scan(&num); err != nil {
			continue
		}
		seg := num[strings.LastIndex(num, "-")+1:]
		if n, err := strconv.ParseInt(seg, 10, 64); err == nil && n > max {
			max = n
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s%03d", prefix, max+1), nil
}

// CreateQuotation menyimpan penawaran DRAFT beserta rincian. Nomor selalu
// otomatis (tidak ada input user) agar unik per kantor-tahun.
func (s *Store) CreateQuotation(officeID int64, requestID *int64, validUntil, notes string, lines []Line, createdBy *int64) (int64, error) {
	if len(lines) == 0 {
		return 0, errors.New("penawaran membutuhkan minimal satu rincian")
	}
	for _, l := range lines {
		if l.Qty <= 0 || l.UnitPrice < 0 {
			return 0, errors.New("qty/harga baris tidak valid")
		}
	}
	var dueVal any
	if strings.TrimSpace(validUntil) != "" {
		dueVal = strings.TrimSpace(validUntil)
	}
	// Retry 2x untuk bentrok nomor (paralel): hitung ulang MAX+1 tiap percobaan.
	var id int64
	for attempt := 0; attempt < 2; attempt++ {
		number, err := s.nextQuotationNumber(officeID)
		if err != nil {
			return 0, err
		}
		id, err = s.insertQuotation(officeID, requestID, number, dueVal, notes, lines, createdBy)
		if err == nil {
			return id, nil
		}
		if isDupErr(err) && attempt == 0 {
			continue
		}
		if isDupErr(err) {
			return 0, errors.New("bentrok nomor penawaran, silakan coba lagi")
		}
		return 0, err
	}
	return 0, errors.New("gagal membuat penawaran setelah 2 percobaan")
}

func (s *Store) insertQuotation(officeID int64, requestID *int64, number string, dueVal any, notes string, lines []Line, createdBy *int64) (int64, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	res, err := tx.Exec(`INSERT INTO quotations(office_id, request_id, number, valid_until, status, notes, created_by)
		VALUES(?,?,?,?,'DRAFT',?,?)`, officeID, requestID, number, dueVal, notes, createdBy)
	if err != nil {
		return 0, err
	}
	id, _ := res.LastInsertId()
	for _, l := range lines {
		lt, _ := money.Mul(l.UnitPrice, l.Qty)
		if _, err := tx.Exec(`INSERT INTO quotation_lines(quotation_id, description, qty, unit_price, line_total)
			VALUES(?,?,?,?,?)`, id, l.Description, l.Qty, l.UnitPrice, lt); err != nil {
			return 0, err
		}
	}
	return id, tx.Commit()
}

// ReqID mengembalikan RequestID ter-deref (0 bila tanpa permohonan) agar
// aman dicetak di template (pointer dicetak sebagai alamat bila langsung).
func (q Quotation) ReqID() int64 {
	if q.RequestID == nil {
		return 0
	}
	return *q.RequestID
}
// quotationTotal menjumlahkan rincian (tanpa PPN: penawaran adalah estimasi
// sebelum pajak; PPN 11% dihitung saat konversi invoice via money.PPN).
func quotationTotal(lines []QuotationLine) int64 {
	var t int64
	for _, l := range lines {
		t = money.Add(t, l.LineTotal)
	}
	return t
}

// GetQuotation mengambil satu penawaran office-scoped beserta rincian.
func (s *Store) GetQuotation(officeID, id int64) (Quotation, error) {
	var q Quotation
	var reqID sql.NullInt64
	var due sql.NullString
	var notes sql.NullString
	if err := s.db.QueryRow(`SELECT id, office_id, request_id, number, valid_until, status, notes
		FROM quotations WHERE id=? AND office_id=?`, id, officeID).
		Scan(&q.ID, &q.OfficeID, &reqID, &q.Number, &due, &q.Status, &notes); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return q, errors.New("penawaran tidak ditemukan")
		}
		return q, err
	}
	if reqID.Valid {
		q.RequestID = &reqID.Int64
	}
	if due.Valid {
		q.ValidUntil = due.String
	}
	if notes.Valid {
		q.Notes = notes.String
	}
	rows, err := s.db.Query(`SELECT id, quotation_id, description, qty, unit_price, line_total
		FROM quotation_lines WHERE quotation_id=? ORDER BY id`, id)
	if err != nil {
		return q, err
	}
	defer rows.Close()
	for rows.Next() {
		var l QuotationLine
		var up, lt string
		if err := rows.Scan(&l.ID, &l.QuotationID, &l.Description, &l.Qty, &up, &lt); err != nil {
			return q, err
		}
		l.UnitPrice, l.LineTotal = deci(up), deci(lt)
		q.Lines = append(q.Lines, l)
	}
	if err := rows.Err(); err != nil {
		return q, err
	}
	q.Total = quotationTotal(q.Lines)
	return q, nil
}

// ExpireOverdueQuotations menandai SENT yang lewat valid_until menjadi
// EXPIRED (office-scoped, satu UPDATE bulk). Hanya SENT yang disentuh —
// DRAFT dibiarkan agar keputusan manual tak tertimpa otomatisasi, dan
// transisi SENT->EXPIRED valid menurut quotTransactions. Audit dicatat
// agregat (satu baris) oleh pemanggil bila n > 0, bukan per baris.
func (s *Store) ExpireOverdueQuotations(officeID int64) (int64, error) {
	res, err := s.db.Exec(`UPDATE quotations SET status='EXPIRED'
		WHERE office_id=? AND status='SENT' AND valid_until IS NOT NULL AND valid_until < CURDATE()`, officeID)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	return n, nil
}

// ListQuotations mendaftar penawaran terbaru (maks 200) beserta totalnya.
func (s *Store) ListQuotations(officeID int64) ([]Quotation, error) {
	rows, err := s.db.Query(`SELECT id, office_id, request_id, number, valid_until, status, notes
		FROM quotations WHERE office_id=? ORDER BY id DESC LIMIT 200`, officeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Quotation
	for rows.Next() {
		var q Quotation
		var reqID sql.NullInt64
		var due, notes sql.NullString
		if err := rows.Scan(&q.ID, &q.OfficeID, &reqID, &q.Number, &due, &q.Status, &notes); err != nil {
			return nil, err
		}
		if reqID.Valid {
			q.RequestID = &reqID.Int64
		}
		if due.Valid {
			q.ValidUntil = due.String
		}
		if notes.Valid {
			q.Notes = notes.String
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		var t sql.NullString
		_ = s.db.QueryRow(`SELECT SUM(line_total) FROM quotation_lines WHERE quotation_id=?`, out[i].ID).Scan(&t)
		if t.Valid {
			out[i].Total = deci(t.String)
		}
	}
	return out, nil
}

// QuotationsForRequest mendaftar penawaran satu permohonan (untuk kartu di request_detail).
func (s *Store) QuotationsForRequest(officeID, requestID int64) ([]Quotation, error) {
	rows, err := s.db.Query(`SELECT id, office_id, request_id, number, valid_until, status, notes
		FROM quotations WHERE office_id=? AND request_id=? ORDER BY id DESC`, officeID, requestID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Quotation
	for rows.Next() {
		var q Quotation
		var reqID sql.NullInt64
		var due, notes sql.NullString
		if err := rows.Scan(&q.ID, &q.OfficeID, &reqID, &q.Number, &due, &q.Status, &notes); err != nil {
			return nil, err
		}
		if reqID.Valid {
			q.RequestID = &reqID.Int64
		}
		if due.Valid {
			q.ValidUntil = due.String
		}
		if notes.Valid {
			q.Notes = notes.String
		}
		out = append(out, q)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	for i := range out {
		var t sql.NullString
		_ = s.db.QueryRow(`SELECT SUM(line_total) FROM quotation_lines WHERE quotation_id=?`, out[i].ID).Scan(&t)
		if t.Valid {
			out[i].Total = deci(t.String)
		}
	}
	return out, nil
}

// SetQuotationStatus memajukan status penawaran manual (ACCEPTED dikecualikan —
// hanya ConvertToInvoice yang boleh menandai ACCEPTED).
func (s *Store) SetQuotationStatus(officeID, id int64, to string) error {
	to = strings.ToUpper(strings.TrimSpace(to))
	if to == "ACCEPTED" {
		return errors.New("ACCEPTED hanya via konversi ke invoice (tombol Setujui)")
	}
	var from string
	if err := s.db.QueryRow(`SELECT status FROM quotations WHERE id=? AND office_id=?`, id, officeID).Scan(&from); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("penawaran tidak ditemukan")
		}
		return err
	}
	ok := false
	for _, t := range quotTransitions[from] {
		if t == to {
			ok = true
			break
		}
	}
	if !ok {
		return fmt.Errorf("transisi %s -> %s tidak diizinkan", from, to)
	}
	res, err := s.db.Exec(`UPDATE quotations SET status=? WHERE id=? AND office_id=?`, to, id, officeID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n != 1 {
		return errors.New("penawaran tidak ditemukan")
	}
	return nil
}

// ConvertToInvoice mengubah penawaran DRAFT/SENT menjadi invoice ISSUED.
//
// KEPUTUSAN DESAIN (didokumentasikan jujur):
//   - Invoice hasil konversi berstatus ISSUED (tagihan resmi langsung), BUKAN
//     DRAFT. Alasan: kode CreateInvoice existing selalu menulis 'ISSUED'
//     (ENUM DRAFT tidak pernah dipakai jalur mana pun); membuat status DRAFT
//     khusus konversi justru menambah cabang status yang tak dikenal
//     anti-overpay/VoidInvoice. Anti-overpay tetap utuh: paid=0 saat lahir,
//     RecordPayment menolak kelebihan bayar seperti biasa.
//   - Tidak memanggil CreateInvoiceExisting agar seluruh konversi (invoice +
//     taut quotation_id + ACCEPTED) atomik dalam SATU transaksi; ~15 baris
//     matematika subtotal/PPN diduplikasi dari CreateInvoice (risiko divergensi
//     dicatat; bila CreateInvoice berubah rumus, samakan di sini).
//   - Nomor invoice diturunkan dari nomor penawaran (Q-.. -> INV-..) + retry
//     suffix bila bentrok uq_invoice_no.
func (s *Store) ConvertToInvoice(officeID, id int64) (int64, error) {
	for attempt := 0; attempt < 3; attempt++ {
		invID, retry, err := s.convertOnce(officeID, id, attempt)
		if err == nil {
			return invID, nil
		}
		if retry && attempt < 2 {
			continue
		}
		return 0, err
	}
	return 0, errors.New("konversi gagal setelah 3 percobaan — coba lagi")
}

func (s *Store) convertOnce(officeID, id int64, attempt int) (int64, bool, error) {
	tx, err := s.db.Begin()
	if err != nil {
		return 0, false, err
	}
	defer tx.Rollback()
	var q Quotation
	var reqID sql.NullInt64
	var due sql.NullString
	if err := tx.QueryRow(`SELECT id, office_id, request_id, number, valid_until, status
		FROM quotations WHERE id=? AND office_id=? FOR UPDATE`, id, officeID).
		Scan(&q.ID, &q.OfficeID, &reqID, &q.Number, &due, &q.Status); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, false, errors.New("penawaran tidak ditemukan")
		}
		return 0, false, err
	}
	if q.Status != "DRAFT" && q.Status != "SENT" {
		return 0, false, fmt.Errorf("hanya penawaran DRAFT/SENT yang dapat disetujui (status: %s)", q.Status)
	}
	if !reqID.Valid {
		return 0, false, errors.New("penawaran tanpa permohonan tidak dapat dikonversi (klien tak diketahui)")
	}
	q.RequestID = &reqID.Int64
	if due.Valid {
		q.ValidUntil = due.String
		// Kedaluwarsa otomatis: lewat valid_until => tandai EXPIRED, tolak konversi.
		var past int
		_ = tx.QueryRow(`SELECT ? > CURDATE()`, due.String).Scan(&past)
		if past == 1 {
			_, _ = tx.Exec(`UPDATE quotations SET status='EXPIRED' WHERE id=?`, id)
			_ = tx.Commit()
			return 0, false, errors.New("penawaran sudah kedaluwarsa (ditandai EXPIRED)")
		}
	}
	// Klien diambil dari permohonan (satu kantor; permohonan milik klien itu).
	var clientID, roid int64
	if err := tx.QueryRow(`SELECT client_id, office_id FROM service_requests WHERE id=?`, *q.RequestID).
		Scan(&clientID, &roid); err != nil {
		return 0, false, errors.New("permohonan penawaran tidak ditemukan")
	}
	if roid != officeID {
		return 0, false, errors.New("permohonan beda kantor")
	}
	rows, err := tx.Query(`SELECT description, qty, unit_price, line_total FROM quotation_lines WHERE quotation_id=? ORDER BY id`, id)
	if err != nil {
		return 0, false, err
	}
	var lines []Line
	for rows.Next() {
		var desc string
		var qty int
		var up, lt string
		if err := rows.Scan(&desc, &qty, &up, &lt); err != nil {
			rows.Close()
			return 0, false, err
		}
		lines = append(lines, Line{Description: desc, Qty: qty, UnitPrice: deci(up)})
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return 0, false, err
	}
	if len(lines) == 0 {
		return 0, false, errors.New("penawaran tanpa rincian tidak dapat dikonversi")
	}
	var sub int64
	for _, l := range lines {
		lt, err := money.Mul(l.UnitPrice, l.Qty)
		if err != nil {
			return 0, false, err
		}
		sub = money.Add(sub, lt)
	}
	tax := money.PPN(sub, 1100) - sub
	total := money.Add(sub, tax)
	// Nomor invoice dari nomor penawaran; suffix -2, -3 bila bentrok.
	invNum := "INV-" + strings.TrimPrefix(q.Number, "Q-")
	if attempt > 0 {
		invNum += "-" + strconv.Itoa(attempt+1)
	}
	res, err := tx.Exec(`INSERT INTO invoices(office_id, client_id, request_id, quotation_id, number, subtotal, tax, total, status)
		VALUES(?,?,?,?,?,?,?,?,'ISSUED')`, officeID, clientID, q.RequestID, id, invNum, sub, tax, total)
	if err != nil {
		if isDupErr(err) {
			return 0, true, err // retry: nomor invoice bentrok
		}
		return 0, false, err
	}
	invID, _ := res.LastInsertId()
	for _, l := range lines {
		lt, _ := money.Mul(l.UnitPrice, l.Qty)
		if _, err := tx.Exec(`INSERT INTO invoice_lines(invoice_id, description, qty, unit_price, line_total)
			VALUES(?,?,?,?,?)`, invID, l.Description, l.Qty, l.UnitPrice, lt); err != nil {
			return 0, false, err
		}
	}
	if _, err := tx.Exec(`UPDATE quotations SET status='ACCEPTED' WHERE id=?`, id); err != nil {
		return 0, false, err
	}
	return invID, false, tx.Commit()
}
