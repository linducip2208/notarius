// Test integrasi quotation (migrasi 030) + serah akta (migrasi 031).
// Mengikuti pola TestMain SKIP yang sama: tanpa NOTARYOS_TEST_DSN,
// seluruh paket diskip oleh TestMain di integration_test.go (BUKAN hijau).
package integration

import (
	"database/sql"
	"testing"
	"time"

	"notaryos/internal/clients"
	"notaryos/internal/deeds"
	"notaryos/internal/finance"
	"notaryos/internal/requests"
)

func TestQuotationExpireAndConvertWithHandover(t *testing.T) {
	clean(t)

	// Seed layanan khusus TEST-QT; IGNORE agar idempoten antar run.
	if _, err := db.Exec(`INSERT IGNORE INTO services(code, category, name) VALUES('TEST-QT','other','Layanan Uji Quotation')`); err != nil {
		t.Fatal(err)
	}
	var sid int64
	if err := db.QueryRow(`SELECT id FROM services WHERE code='TEST-QT'`).Scan(&sid); err != nil {
		t.Skip("seed services TEST-QT tidak tersedia di test db")
	}

	office := seedOffice(t)
	cid, err := clients.NewStore(db).Create(clients.Client{OfficeID: office, Kind: "individual", FullName: "QT Tester", Phone: "0888000099"})
	if err != nil {
		t.Fatal(err)
	}
	rid, err := requests.NewStore(db).Create(office, cid, sid, "Permohonan QT", "", nil)
	if err != nil {
		t.Fatal(err)
	}

	fs := finance.NewStore(db)
	future := time.Now().AddDate(0, 0, 7).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// --- Jalur 1: DRAFT (2 lines) -> SENT -> EXPIRED ---
	q1, err := fs.CreateQuotation(office, &rid, future, "uji expire", []finance.Line{
		{Description: "Jasa Akta", Qty: 1, UnitPrice: 2000000},
		{Description: "Biaya PNBP", Qty: 2, UnitPrice: 500000},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	got, err := fs.GetQuotation(office, q1)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != "DRAFT" || len(got.Lines) != 2 {
		t.Fatalf("quotation awal salah: status=%q lines=%d, harus DRAFT/2", got.Status, len(got.Lines))
	}
	if want := int64(3000000); got.Total != want {
		t.Fatalf("total quotation salah: %d, harus %d", got.Total, want)
	}
	if err := fs.SetQuotationStatus(office, q1, "SENT"); err != nil {
		t.Fatal(err)
	}
	// Buat lewat tanggal via UPDATE (tanpa API pemalsu tanggal).
	if _, err := db.Exec(`UPDATE quotations SET valid_until=? WHERE id=?`, yesterday, q1); err != nil {
		t.Fatal(err)
	}
	n, err := fs.ExpireOverdueQuotations(office)
	if err != nil {
		t.Fatal(err)
	}
	if n < 1 {
		t.Fatalf("ExpireOverdueQuotations menyentuh %d baris, harus >=1", n)
	}
	if got, err := fs.GetQuotation(office, q1); err != nil {
		t.Fatal(err)
	} else if got.Status != "EXPIRED" {
		t.Fatalf("status q1=%q, harus EXPIRED", got.Status)
	}

	// --- Jalur 2: quotation baru -> convert -> invoice ISSUED + quotation ACCEPTED ---
	q2, err := fs.CreateQuotation(office, &rid, future, "uji convert", []finance.Line{
		{Description: "Jasa Akta", Qty: 1, UnitPrice: 1000000},
		{Description: "Legalisasi", Qty: 1, UnitPrice: 500000},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	invID, err := fs.ConvertToInvoice(office, q2)
	if err != nil {
		t.Fatal(err)
	}
	var invStatus string
	var invQuot sql.NullInt64
	if err := db.QueryRow(`SELECT status, quotation_id FROM invoices WHERE id=? AND office_id=?`, invID, office).Scan(&invStatus, &invQuot); err != nil {
		t.Fatal(err)
	}
	if invStatus != "ISSUED" {
		t.Fatalf("status invoice=%q, harus ISSUED", invStatus)
	}
	if !invQuot.Valid || invQuot.Int64 != q2 {
		t.Fatalf("invoice.quotation_id=%+v, harus %d", invQuot, q2)
	}
	if got, err := fs.GetQuotation(office, q2); err != nil {
		t.Fatal(err)
	} else if got.Status != "ACCEPTED" {
		t.Fatalf("status q2=%q, harus ACCEPTED", got.Status)
	}

	// --- Serah akta (migrasi 031): riwayat deed_handovers via SQL mentah ---
	deedID, err := deeds.NewStore(db).CreateDraft(office, &rid, "Akta Uji Serah", "PT", nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := db.Exec(`INSERT INTO deed_handovers(office_id, deed_id, handed_to, handed_at, note)
		VALUES(?,?,?,UTC_TIMESTAMP(),?)`, office, deedID, "Penerima Uji", "serah minuta")
	if err != nil {
		t.Fatal(err)
	}
	hid, _ := res.LastInsertId()
	var to, note string
	var cnt int
	if err := db.QueryRow(`SELECT handed_to, COALESCE(note,''), COUNT(*) OVER () FROM deed_handovers WHERE id=? AND deed_id=? AND office_id=?`, hid, deedID, office).Scan(&to, &note, &cnt); err != nil {
		t.Fatal(err)
	}
	if to != "Penerima Uji" || cnt != 1 {
		t.Fatalf("handover salah: to=%q cnt=%d", to, cnt)
	}
	if note != "serah minuta" {
		t.Fatalf("note handover=%q, harus serah minuta", note)
	}

	// Cleanup targeted urutan FK terbalik (service TEST-QT dibiarkan, seed IGNORE).
	for _, q := range []struct {
		query string
		args  []any
	}{
		{`DELETE FROM deed_handovers WHERE id=?`, []any{hid}},
		{`DELETE FROM deed_versions WHERE deed_id=?`, []any{deedID}},
		{`DELETE FROM deed_drafts WHERE id=?`, []any{deedID}},
		{`DELETE FROM payments WHERE invoice_id=?`, []any{invID}},
		{`DELETE FROM invoice_lines WHERE invoice_id=?`, []any{invID}},
		{`DELETE FROM invoices WHERE id=?`, []any{invID}},
		{`DELETE FROM quotation_lines WHERE quotation_id=?`, []any{q2}},
		{`DELETE FROM quotation_lines WHERE quotation_id=?`, []any{q1}},
		{`DELETE FROM quotations WHERE id=?`, []any{q2}},
		{`DELETE FROM quotations WHERE id=?`, []any{q1}},
		{`DELETE FROM request_events WHERE request_id=?`, []any{rid}},
		{`DELETE FROM service_requests WHERE id=?`, []any{rid}},
		{`DELETE FROM clients WHERE id=?`, []any{cid}},
		{`DELETE FROM offices WHERE id=?`, []any{office}},
	} {
		if _, err := db.Exec(q.query, q.args...); err != nil {
			t.Fatal("cleanup:", err)
		}
	}
}
