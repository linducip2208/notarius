//go:build ignore

// generate_dummy membuat internal/ocr/testdata/dummy_ktp.png — gambar KTP
// PALSU yang jelas-jelas dummy (nama "CONTOH SAJA", NIK nol semua) untuk
// pengujian AssessImage + BuildVariants TANPA Tesseract/DB.
//
// Cara pakai: go run generate_dummy.go   (dari direktori testdata/)
// Font: bitmap 3x5 buatan sendiri, stdlib saja (image/png). Tesseract tidak
// diharapkan mampu membaca font balok ini — dan memang tidak perlu: fixture
// ini HANYA untuk quality-gate + preprocessing, bukan akurasi OCR.
// Akurasi parsing diuji dari konstanta teks di ktp_parse_test.go.
package main

import (
	"image"
	"image/color"
	"image/png"
	"log"
	"os"
)

var font = map[rune][]string{
	'A': {"#.#", "#.#", "###", "#.#", "#.#"},
	'B': {"##.", "#.#", "##.", "#.#", "##."},
	'C': {".##", "#..", "#..", "#..", ".##"},
	'D': {"##.", "#.#", "#.#", "#.#", "##."},
	'E': {"###", "#..", "##.", "#..", "###"},
	'F': {"###", "#..", "##.", "#..", "#.."},
	'H': {"#.#", "#.#", "###", "#.#", "#.#"},
	'I': {"###", ".#.", ".#.", ".#.", "###"},
	'J': {"..#", "..#", "..#", "#.#", ".#."},
	'K': {"#.#", "#.#", "##.", "#.#", "#.#"},
	'L': {"#..", "#..", "#..", "#..", "###"},
	'M': {"#.#", "###", "#.#", "#.#", "#.#"},
	'N': {"#.#", "##.", "#.#", ".##", "#.#"},
	'O': {".#.", "#.#", "#.#", "#.#", ".#."},
	'P': {"##.", "#.#", "##.", "#..", "#.."},
	'R': {"##.", "#.#", "##.", "#.#", "#.#"},
	'S': {".##", "#..", ".#.", "..#", "##."},
	'T': {"###", ".#.", ".#.", ".#.", ".#."},
	'U': {"#.#", "#.#", "#.#", "#.#", "###"},
	'Y': {"#.#", "#.#", ".#.", ".#.", ".#."},
	'0': {"###", "#.#", "#.#", "#.#", "###"},
	' ': {"...", "...", "...", "...", "..."},
	':': {"...", ".#.", "...", ".#.", "..."},
	'-': {"...", "...", "###", "...", "..."},
	'.': {"...", "...", "...", "...", ".#."},
	'/': {"..#", "..#", ".#.", "#..", "#.."},
}

func drawText(img *image.RGBA, x, y, scale int, col color.Color, s string) {
	cx := x
	for _, r := range s {
		g, ok := font[r]
		if !ok {
			g = font[' ']
		}
		for gy, row := range g {
			for gx, c := range row {
				if c != '#' {
					continue
				}
				for dy := 0; dy < scale; dy++ {
					for dx := 0; dx < scale; dx++ {
						img.Set(cx+gx*scale+dx, y+gy*scale+dy, col)
					}
				}
			}
		}
		cx += 4 * scale
	}
}

func fillRect(img *image.RGBA, x, y, w, h int, col color.Color) {
	for j := y; j < y+h; j++ {
		for i := x; i < x+w; i++ {
			img.Set(i, j, col)
		}
	}
}

func hline(img *image.RGBA, x, y, w, t int, col color.Color) {
	fillRect(img, x, y-t/2, w, t, col)
}

func main() {
	const W, H = 1200, 760
	img := image.NewRGBA(image.Rect(0, 0, W, H))
	bg := color.RGBA{150, 175, 200, 255}
	navy := color.RGBA{25, 55, 110, 255}
	ink := color.RGBA{20, 30, 60, 255}
	white := color.RGBA{255, 255, 255, 255}
	black := color.RGBA{0, 0, 0, 255}
	red := color.RGBA{150, 30, 30, 255}

	fillRect(img, 0, 0, W, H, bg)
	// Bingkai luar tebal agar Laplacian variance (blur) tinggi & stabil.
	for t := 0; t < 6; t++ {
		hline(img, 0, t, W, 1, black)
		hline(img, 0, H-1-t, W, 1, black)
		for y := 0; y < H; y++ {
			img.Set(t, y, black)
			img.Set(W-1-t, y, black)
		}
	}
	// Pita header + footer.
	fillRect(img, 6, 6, W-12, 96, navy)
	drawText(img, 48, 30, 6, white, "KTP - CONTOH DUMMY")
	fillRect(img, 6, H-102, W-12, 96, navy)
	drawText(img, 48, H-78, 4, white, "DUMMY - BUKAN DOKUMEN ASLI")

	// Kolom teks dummy kiri.
	drawText(img, 60, 170, 5, ink, "NIK : 0000000000000000")
	drawText(img, 60, 240, 5, ink, "NAMA : CONTOH SAJA")
	drawText(img, 60, 310, 5, ink, "LAHIR : 00-00-0000")
	drawText(img, 60, 380, 5, ink, "ALAMAT : JALAN CONTOH")
	drawText(img, 60, 450, 5, red, "BUKAN KTP ASLI")

	// Kotak foto placeholder kanan (tajam: bantu kontras + blur).
	px, py, pw, ph := 880, 150, 260, 320
	fillRect(img, px, py, pw, ph, color.RGBA{205, 205, 205, 255})
	for t := 0; t < 4; t++ {
		hline(img, px, py+t, pw, 1, black)
		hline(img, px, py+ph-1-t, pw, 1, black)
		for y := py; y < py+ph; y++ {
			img.Set(px+t, y, black)
			img.Set(px+pw-1-t, y, black)
		}
	}
	// Silang diagonal.
	for i := 0; i < pw-8; i++ {
		j := i * (ph - 8) / (pw - 8)
		for t := -2; t <= 2; t++ {
			img.Set(px+4+i, py+4+j+t, color.RGBA{90, 90, 90, 255})
			img.Set(px+4+i, py+ph-5-j+t, color.RGBA{90, 90, 90, 255})
		}
	}
	drawText(img, px+78, py+ph+24, 4, ink, "FOTO")

	f, err := os.Create("dummy_ktp.png")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		log.Fatal(err)
	}
	log.Print("dummy_ktp.png written (KTP PALSU — CONTOH SAJA, NIK 000...)")
}
