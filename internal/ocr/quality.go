package ocr

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"os"
)

// QualityReport is the enterprise gate BEFORE OCR runs.
// Never blocks silently: low grade returns actionable Issues.
type QualityReport struct {
	Width, Height int
	Brightness    float64 // 0..255 mean
	Contrast      float64 // stddev
	Blur          float64 // variance of Laplacian (higher = sharper)
	Grade         string  // A_EXCELLENT | B_GOOD | C_RETRY_RECOMMENDED | D_RESHOOT_REQUIRED
	Score         float64 // 0..100
	Issues        []string
	Reshoot       bool
}

func AssessImage(path string) (QualityReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return QualityReport{}, err
	}
	defer f.Close()
	img, _, err := image.Decode(f)
	if err != nil {
		return QualityReport{}, fmt.Errorf("decode gagal (format harus JPG/PNG): %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	// downscale sampling max 600px sisi panjang agar cepat
	stepX, stepY := 1, 1
	if w > 600 {
		stepX = w / 600
	}
	if h > 600 {
		stepY = h / 600
	}
	var sum, sum2 float64
	var n float64
	gray := func(x, y int) float64 {
		r, g, bl, _ := img.At(x+b.Min.X, y+b.Min.Y).RGBA()
		// 0..255
		return (0.299*float64(r>>8) + 0.587*float64(g>>8) + 0.114*float64(bl>>8))
	}
	// mean + stddev
	for y := 0; y < h; y += stepY {
		for x := 0; x < w; x += stepX {
			v := gray(x, y)
			sum += v
			sum2 += v * v
			n++
		}
	}
	mean := sum / n
	variance := sum2/n - mean*mean
	contrast := math.Sqrt(math.Max(variance, 0))
	// Laplacian variance (blur): L = 4c - up - down - left - right
	var lsum, lsum2 float64
	var ln float64
	for y := stepY; y < h-stepY; y += stepY {
		for x := stepX; x < w-stepX; x += stepX {
			c := gray(x, y)
			l := 4*c - gray(x, y-stepY) - gray(x, y+stepY) - gray(x-stepX, y) - gray(x+stepX, y)
			lsum += l
			lsum2 += l * l
			ln++
		}
	}
	lmean := lsum / math.Max(ln, 1)
	blur := lsum2/math.Max(ln, 1) - lmean*lmean

	rep := QualityReport{Width: w, Height: h, Brightness: mean, Contrast: contrast, Blur: blur}
	score := 100.0
	// Resolusi: KTP butuh >= 1000px lebar untuk 16 digit terbaca stabil
	if w < 800 {
		score -= 30
		rep.Issues = append(rep.Issues, fmt.Sprintf("resolusi rendah (%dx%d) — dekatkan kamera / pakai 12MP+", w, h))
	} else if w < 1200 {
		score -= 10
		rep.Issues = append(rep.Issues, "resolusi pas-pasan — usahakan kartu memenuhi 70% bingkai")
	}
	if mean < 60 || mean > 210 {
		score -= 25
		if mean < 60 {
			rep.Issues = append(rep.Issues, "terlalu gelap — tambah cahaya merata, matikan flash langsung")
		} else {
			rep.Issues = append(rep.Issues, "terlalu terang / glare — miringkan kartu 10°, hindari pantulan")
		}
	} else if mean < 80 || mean > 190 {
		score -= 8
		rep.Issues = append(rep.Issues, "pencahayaan kurang merata")
	}
	if contrast < 25 {
		score -= 25
		rep.Issues = append(rep.Issues, "kontras rendah (buram/pudar) — fokuskan, bersihkan lensa")
	} else if contrast < 35 {
		score -= 8
		rep.Issues = append(rep.Issues, "kontras sedang — pastikan tidak miring/berbayang")
	}
	if blur < 25 {
		score -= 30
		rep.Issues = append(rep.Issues, "foto blur (tangan goyang) — tahan napas, sandarkan HP, ketuk fokus ke NIK")
	} else if blur < 80 {
		score -= 10
		rep.Issues = append(rep.Issues, "ketajaman kurang — ketuk fokus ke teks NIK sebelum jepret")
	}
	if score < 0 {
		score = 0
	}
	rep.Score = math.Round(score)
	switch {
	case score >= 85:
		rep.Grade = "A_EXCELLENT"
	case score >= 65:
		rep.Grade = "B_GOOD"
	case score >= 45:
		rep.Grade = "C_RETRY_RECOMMENDED"
		rep.Reshoot = false
	default:
		rep.Grade = "D_RESHOOT_REQUIRED"
		rep.Reshoot = true
	}
	return rep, nil
}
