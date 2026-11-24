package app

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestSquareCropPNG(t *testing.T) {
	// Kanvas 10x6 merah dengan 1 piksel hijau di tengah.
	src := image.NewRGBA(image.Rect(0, 0, 10, 6))
	for y := 0; y < 6; y++ {
		for x := 0; x < 10; x++ {
			src.Set(x, y, color.RGBA{R: 200, A: 255})
		}
	}
	src.Set(5, 3, color.RGBA{G: 200, A: 255})
	var buf bytes.Buffer
	if err := png.Encode(&buf, src); err != nil {
		t.Fatal(err)
	}
	out, ok := squareCropPNG(buf.Bytes())
	if !ok {
		t.Fatal("crop harus berhasil untuk PNG valid")
	}
	got, _, err := image.Decode(bytes.NewReader(out))
	if err != nil {
		t.Fatal(err)
	}
	if b := got.Bounds(); b.Dx() != 6 || b.Dy() != 6 {
		t.Fatalf("hasil harus 6x6, dapat %dx%d", b.Dx(), b.Dy())
	}
	if _, ok := squareCropPNG([]byte("bukan gambar")); ok {
		t.Fatal("input rusak wajib ok=false (fail-open)")
	}
	if _, ok := squareCropPNG(nil); ok {
		t.Fatal("input kosong wajib ok=false")
	}
}
