package deliver

import (
	"strings"
	"testing"
)

func TestWALink(t *testing.T) {
	if got := NormalizePhoneForWA("0812-3456-789"); got != "628123456789" {
		t.Fatalf("normalisasi 08 salah: %q", got)
	}
	if got := NormalizePhoneForWA("+62 812 3456 789"); got != "628123456789" {
		t.Fatalf("normalisasi +62 salah: %q", got)
	}
	if got := PreferWA("", "0812-000-111"); got != "62812000111" {
		t.Fatalf("fallback ke phone salah: %q", got)
	}
	if got := PreferWA("0819-999-888", "0812-000-111"); got != "62819999888" {
		t.Fatalf("wa_number harus diutamakan: %q", got)
	}
	if WALink("", "halo") != "" || WALink("abc", "halo") != "" {
		t.Fatal("nomor tak valid harus menghasilkan tautan kosong (tombol disembunyikan)")
	}
	link := WALink("0812-3456-789", "Halo Budi, permohonan #1")
	if !strings.HasPrefix(link, "https://wa.me/628123456789?text=") {
		t.Fatalf("tautan wa.me salah: %q", link)
	}
}
