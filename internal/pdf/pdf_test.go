package pdf

import (
	"strings"
	"testing"
)

func TestExportMultipage(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 200; i++ {
		b.WriteString("Baris dokumen akta yang cukup panjang untuk menguji wrapping dan paginasi otomatis.\n")
	}
	out := Export("Akta Uji Panjang", b.String(), true)
	if len(out) == 0 || string(out[:5]) != "%PDF-" {
		t.Fatal("bukan PDF valid")
	}
	s := string(out)
	if !strings.Contains(s, "Halaman 1 dari") {
		t.Fatal("footer halaman hilang")
	}
	if strings.Count(s, "/Type/Page/Parent") < 2 {
		t.Fatal("dokumen panjang harus >1 halaman")
	}
	if !strings.Contains(s, "DRAFT") {
		t.Fatal("watermark DRAFT hilang")
	}
}
