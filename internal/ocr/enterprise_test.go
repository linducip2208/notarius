package ocr

import "testing"

func TestAssessImage_Sample(t *testing.T) {
	q, err := AssessImage(`D:\folder selamat lindu\download\Scan KTP.JPG`)
	if err != nil {
		t.Skipf("sampel tidak ada di CI: %v", err)
	}
	if q.Width < 500 {
		t.Fatalf("width terlalu kecil: %d", q.Width)
	}
	if q.Grade == "" || q.Score == 0 {
		t.Fatal("grade/score kosong")
	}
	t.Logf("grade=%s score=%.0f issues=%v", q.Grade, q.Score, q.Issues)
}

func TestBuildVariants(t *testing.T) {
	v, cleanup, err := BuildVariants(`D:\folder selamat lindu\download\Scan KTP.JPG`)
	defer cleanup()
	if err != nil {
		t.Skipf("sampel tidak ada: %v", err)
	}
	if len(v) != 3 {
		t.Fatalf("varian = %d, mau 3", len(v))
	}
}
