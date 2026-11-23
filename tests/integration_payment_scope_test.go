// Test isolasi office-scope untuk RecordPayment: membayar invoice kantor A
// dari konteks kantor B harus gagal dan tidak meninggalkan payment.
package integration

import (
	"testing"

	"notaryos/internal/clients"
	"notaryos/internal/finance"
)

func TestFinancePaymentOfficeIsolation(t *testing.T) {
	clean(t)
	officeA := seedOffice(t)
	officeB := seedOffice(t)
	cs, fs := clients.NewStore(db), finance.NewStore(db)
	cid, err := cs.Create(clients.Client{OfficeID: officeA, Kind: "individual", FullName: "Isolasi", Phone: "0899000099"})
	if err != nil {
		t.Fatal(err)
	}
	invid, err := fs.CreateInvoice(officeA, cid, nil, "INV-ISOLASI", []finance.Line{
		{Description: "Jasa", Qty: 1, UnitPrice: 1000000},
	}, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	// Bayar dari kantor lain harus gagal.
	if err := fs.RecordPayment(officeB, invid, 500000, "CASH", "X-B", nil); err == nil {
		t.Fatal("pembayaran lintas kantor harus gagal")
	}
	// Tidak boleh ada payment yang tercatat.
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM payments WHERE invoice_id=?`, invid).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 0 {
		t.Fatalf("payment lintas kantor tercatat: %d", n)
	}
	// Invoice kantor A tetap belum terbayar.
	var status string
	var paid string
	if err := db.QueryRow(`SELECT status, paid FROM invoices WHERE id=?`, invid).Scan(&status, &paid); err != nil {
		t.Fatal(err)
	}
	if status != "ISSUED" {
		t.Fatalf("status berubah oleh kantor lain: %s", status)
	}
	// Bayar dari kantor pemilik harus berhasil.
	if err := fs.RecordPayment(officeA, invid, 1000000, "TRANSFER", "R-OK", nil); err != nil {
		t.Fatal("pembayaran kantor pemilik harus berhasil:", err)
	}
}
