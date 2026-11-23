package ocr

// Uji quality-gate + preprocessing terhadap fixture KTP PALSU di testdata/.
// Semua uji di file ini murni fungsi piksel (tanpa Tesseract, tanpa DB)
// sehingga WAJIB lolos di CI. Riwayat: sebelumnya memakai path absolut
// mesin pribadi (D:\folder selamat lindu\...) sehingga SKIP diam-diam di CI.

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func fixturePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller gagal")
	}
	p := filepath.Join(filepath.Dir(file), "testdata", "dummy_ktp.png")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("fixture hilang (%s): regenerasi via go run ./testdata/generate_dummy.go: %v", p, err)
	}
	return p
}

func TestAssessImage_Fixture(t *testing.T) {
	q, err := AssessImage(fixturePath(t))
	if err != nil {
		t.Fatalf("AssessImage fixture gagal: %v", err)
	}
	if q.Width != 1200 || q.Height != 760 {
		t.Fatalf("dimensi = %dx%d, mau 1200x760", q.Width, q.Height)
	}
	if q.Grade == "" || q.Score == 0 {
		t.Fatal("grade/score kosong")
	}
	t.Logf("grade=%s score=%.0f brightness=%.1f contrast=%.1f blur=%.1f issues=%v",
		q.Grade, q.Score, q.Brightness, q.Contrast, q.Blur, q.Issues)
}

func TestBuildVariants_Fixture(t *testing.T) {
	v, cleanup, err := BuildVariants(fixturePath(t))
	if err != nil {
		t.Fatalf("BuildVariants fixture gagal: %v", err)
	}
	defer cleanup()
	if len(v) != 3 {
		t.Fatalf("varian = %d, mau 3 (ORIGINAL|CONTRAST_2X|OTSU_2X)", len(v))
	}
	want := []string{"ORIGINAL", "CONTRAST_2X", "OTSU_2X"}
	for i, k := range want {
		if v[i].Kind != k {
			t.Fatalf("varian[%d] = %s, mau %s", i, v[i].Kind, k)
		}
		if _, err := os.Stat(v[i].Path); err != nil {
			t.Fatalf("varian %s hilang di %s: %v", k, v[i].Path, err)
		}
	}
}

func TestBuildVariants_CleanupRemovesTemp(t *testing.T) {
	v, cleanup, err := BuildVariants(fixturePath(t))
	if err != nil {
		t.Fatal(err)
	}
	cleanup()
	// v[0] ORIGINAL menunjuk fixture asli (tetap ada); v[1..] temp wajib hilang.
	if _, err := os.Stat(v[0].Path); err != nil {
		t.Fatalf("fixture asli ikut terhapus: %v", err)
	}
	for _, vv := range v[1:] {
		if _, err := os.Stat(vv.Path); !os.IsNotExist(err) {
			t.Fatalf("temp %s (%s) tidak dibersihkan", vv.Kind, vv.Path)
		}
	}
}

func TestAssessImage_MissingFile(t *testing.T) {
	if _, err := AssessImage(filepath.Join("testdata", "tidak-ada.png")); err == nil {
		t.Fatal("file hilang harus error, bukan diam")
	}
}

func TestBuildVariants_MissingFile(t *testing.T) {
	if _, _, err := BuildVariants(filepath.Join("testdata", "tidak-ada.png")); err == nil {
		t.Fatal("file hilang harus error, bukan diam")
	}
}

// TestTesseractExtract_Fixture adalah uji OCR ujung-ke-ujung yang JUJUR:
// default SKIP dengan alasan eksplisit; hanya jalan bila operator meminta
// eksplisit (NOTARYOS_TEST_TESSERACT=1) DAN biner tesseract + data "ind"
// tersedia. CI tanpa Tesseract tidak pernah diklaim menguji OCR.
func TestTesseractExtract_Fixture(t *testing.T) {
	if os.Getenv("NOTARYOS_TEST_TESSERACT") != "1" {
		t.Skipf("butuh OCR asli (tesseract + data bahasa ind): set NOTARYOS_TEST_TESSERACT=1 untuk menjalankan; lihat docs/TEST_STRATEGY.md")
	}
	bin := os.Getenv("OCR_BINARY")
	if bin == "" {
		bin = "tesseract"
	}
	if _, err := exec.LookPath(bin); err != nil {
		t.Skipf("butuh biner tesseract (%q tidak ditemukan di PATH/OCR_BINARY): %v", bin, err)
	}
	if out, err := exec.Command(bin, "--list-langs").Output(); err != nil || !strings.Contains(string(out), "ind") {
		t.Skipf("butuh data bahasa tesseract 'ind' (tesseract --list-langs): %v out=%q", err, string(out))
	}
	prov, err := NewTesseract(bin)
	if err != nil {
		t.Skipf("NewTesseract gagal (tesseract belum terpasang benar): %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	res, err := prov.Extract(ctx, fixturePath(t))
	if err != nil {
		t.Fatalf("Extract fixture gagal: %v", err)
	}
	// Fixture font-balok tidak dijamin terbaca sebagai teks KTP — yang
	// dijamin: pipeline enterprise tidak crash dan metadata quality terisi.
	if res.Fields["ocr_quality_grade"] == "" {
		t.Fatal("metadata ocr_quality_grade kosong")
	}
	if _, ok := res.Fields["text"]; !ok {
		t.Fatal("Fields[text] hilang")
	}
	t.Logf("variant=%s psm=%s grade=%s text_len=%d",
		res.Fields["ocr_variant"], res.Fields["ocr_psm"],
		res.Fields["ocr_quality_grade"], len(res.Fields["text"]))
}
