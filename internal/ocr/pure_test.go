package ocr

// Uji fungsi murni OCR (tanpa Tesseract, tanpa DB): grade quality-gate,
// fallback non-gambar, klasifikasi nama file, NoopProvider. Selalu jalan di CI.

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func writeTestPNG(t *testing.T, img image.Image) string {
	t.Helper()
	p := filepath.Join(t.TempDir(), "s.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

func solidImage(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func checkerImage(w, h, cell int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x/cell+y/cell)%2 == 0 {
				img.Set(x, y, color.White)
			} else {
				img.Set(x, y, color.Black)
			}
		}
	}
	return img
}

func TestAssessImage_GradeBounds(t *testing.T) {
	// Papan catur besar, terang-sedang, tajam, kontras tinggi → nilai sempurna.
	good := writeTestPNG(t, checkerImage(1200, 700, 20))
	q, err := AssessImage(good)
	if err != nil {
		t.Fatal(err)
	}
	if q.Grade != "A_EXCELLENT" || q.Score != 100 {
		t.Fatalf("checker 1200px harus A/100, dapat %s/%.0f issues=%v", q.Grade, q.Score, q.Issues)
	}
	if q.Reshoot {
		t.Fatal("gambar bagus tidak boleh minta reshoot")
	}

	// Kecil + gelap + datar + blur → skor 0, wajib reshoot.
	bad := writeTestPNG(t, solidImage(100, 60, color.RGBA{30, 30, 30, 255}))
	qb, err := AssessImage(bad)
	if err != nil {
		t.Fatal(err)
	}
	if qb.Grade != "D_RESHOOT_REQUIRED" || !qb.Reshoot {
		t.Fatalf("gambar buruk harus D+reshoot, dapat %s reshoot=%v issues=%v", qb.Grade, qb.Reshoot, qb.Issues)
	}
	if qb.Score != 0 {
		t.Fatalf("skor gambar buruk harus 0, dapat %.0f", qb.Score)
	}
}

func TestBuildVariants_NonImage(t *testing.T) {
	p := filepath.Join(t.TempDir(), "doc.txt")
	if err := os.WriteFile(p, []byte("bukan gambar"), 0o644); err != nil {
		t.Fatal(err)
	}
	v, cleanup, err := BuildVariants(p)
	defer cleanup()
	if err != nil {
		t.Fatal(err)
	}
	if len(v) != 1 || v[0].Kind != "ORIGINAL" || v[0].Path != p {
		t.Fatalf("non-gambar harus 1 varian ORIGINAL as-is, dapat %+v", v)
	}
}

func TestClassify_Table(t *testing.T) {
	cases := map[string]DocType{
		"scan_ktp.jpg":       DocKTP,
		"KTP-ibu.PNG":        DocKTP,
		"kk_keluarga.jpg":    DocKK,
		"kartu_keluarga.png": DocKK,
		"npwp-2024.jpg":      DocNPWP,
		"akta.pdf":           DocOther,
	}
	for name, want := range cases {
		if got := Classify(name); got != want {
			t.Fatalf("Classify(%q) = %s, mau %s", name, got, want)
		}
	}
}

func TestNoopProvider_HonestError(t *testing.T) {
	var p Provider = NoopProvider{Reason: "belum install"}
	if p.Name() != "unconfigured" {
		t.Fatalf("Name = %q", p.Name())
	}
	if _, err := p.Extract(t.Context(), "x.jpg"); err == nil {
		t.Fatal("NoopProvider harus error eksplisit, bukan hasil palsu")
	}
}
