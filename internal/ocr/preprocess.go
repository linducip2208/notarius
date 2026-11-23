package ocr

import (
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"sort"
)

// Variant is one preprocessed image ready for Tesseract.
type Variant struct {
	Path string // temp PNG path (caller removes)
	Kind string // ORIGINAL | CONTRAST_2X | OTSU_2X
}

// BuildVariants creates enterprise preprocessing outputs:
// 2x bilinear upscale + grayscale + autocontrast (+ Otsu binary).
// Pure stdlib — no OpenCV needed.
func BuildVariants(srcPath string) ([]Variant, func(), error) {
	f, err := os.Open(srcPath)
	if err != nil {
		return nil, func() {}, err
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		// bukan gambar (mis. PDF): kembalikan original saja
		return []Variant{{Path: srcPath, Kind: "ORIGINAL"}}, func() {}, nil
	}
	g := toGray(src)
	up := upscale2x(g)
	ac := autocontrast(up)
	padded := padWhite(ac, 24)

	v1, err := writeTempPNG(padded)
	if err != nil {
		return nil, func() {}, err
	}
	bin := otsu(padded)
	paddedBin := padWhite(bin, 24)
	v2, err := writeTempPNG(paddedBin)
	if err != nil {
		os.Remove(v1)
		return nil, func() {}, err
	}
	cleanup := func() { os.Remove(v1); os.Remove(v2) }
	return []Variant{
		{Path: srcPath, Kind: "ORIGINAL"},
		{Path: v1, Kind: "CONTRAST_2X"},
		{Path: v2, Kind: "OTSU_2X"},
	}, cleanup, nil
}

func toGray(src image.Image) *image.Gray {
	b := src.Bounds()
	g := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			g.SetGray(x, y, color.GrayModel.Convert(src.At(x, y)).(color.Gray))
		}
	}
	return g
}

// upscale2x bilinear sederhana.
func upscale2x(g *image.Gray) *image.Gray {
	b := g.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewGray(image.Rect(0, 0, w*2, h*2))
	at := func(x, y int) float64 {
		if x < 0 {
			x = 0
		}
		if y < 0 {
			y = 0
		}
		if x >= w {
			x = w - 1
		}
		if y >= h {
			y = h - 1
		}
		return float64(g.GrayAt(b.Min.X+x, b.Min.Y+y).Y)
	}
	for y := 0; y < h*2; y++ {
		for x := 0; x < w*2; x++ {
			sx := float64(x) / 2
			sy := float64(y) / 2
			x0, y0 := int(math.Floor(sx)), int(math.Floor(sy))
			fx, fy := sx-float64(x0), sy-float64(y0)
			v := at(x0, y0)*(1-fx)*(1-fy) + at(x0+1, y0)*fx*(1-fy) + at(x0, y0+1)*(1-fx)*fy + at(x0+1, y0+1)*fx*fy
			out.SetGray(x, y, color.Gray{Y: uint8(math.Round(v))})
		}
	}
	return out
}

// autocontrast: stretch persentil 2–98 ke 0..255.
func autocontrast(g *image.Gray) *image.Gray {
	b := g.Bounds()
	hist := make([]int, 256)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			hist[g.GrayAt(x, y).Y]++
		}
	}
	total := b.Dx() * b.Dy()
	lo, hi := 0, 255
	acc := 0
	for i := 0; i < 256; i++ {
		acc += hist[i]
		if float64(acc)/float64(total) >= 0.02 {
			lo = i
			break
		}
	}
	acc = 0
	for i := 255; i >= 0; i-- {
		acc += hist[i]
		if float64(acc)/float64(total) >= 0.02 {
			hi = i
			break
		}
	}
	if hi <= lo+8 {
		return g
	}
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			v := (float64(g.GrayAt(x, y).Y) - float64(lo)) * 255 / float64(hi-lo)
			if v < 0 {
				v = 0
			}
			if v > 255 {
				v = 255
			}
			out.SetGray(x, y, color.Gray{Y: uint8(math.Round(v))})
		}
	}
	return out
}

// otsu thresholding untuk cap/stempel & foto miring kontras rendah.
func otsu(g *image.Gray) *image.Gray {
	b := g.Bounds()
	hist := make([]int, 256)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			hist[g.GrayAt(x, y).Y]++
		}
	}
	total := float64(b.Dx() * b.Dy())
	var sum float64
	for i, c := range hist {
		sum += float64(i * c)
	}
	var sumB, wB, best float64
	thresh := 128
	for t := 0; t < 256; t++ {
		wB += float64(hist[t])
		if wB == 0 {
			continue
		}
		wF := total - wB
		if wF == 0 {
			break
		}
		sumB += float64(t * hist[t])
		mB := sumB / wB
		mF := (sum - sumB) / wF
		between := wB * wF * (mB - mF) * (mB - mF)
		if between > best {
			best = between
			thresh = t
		}
	}
	out := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if int(g.GrayAt(x, y).Y) > thresh {
				out.SetGray(x, y, color.Gray{Y: 255})
			} else {
				out.SetGray(x, y, color.Gray{Y: 0})
			}
		}
	}
	return out
}

func padWhite(g *image.Gray, p int) *image.Gray {
	b := g.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewGray(image.Rect(0, 0, w+p*2, h+p*2))
	for y := 0; y < h+p*2; y++ {
		for x := 0; x < w+p*2; x++ {
			out.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			out.SetGray(x+p, y+p, g.GrayAt(b.Min.X+x, b.Min.Y+y))
		}
	}
	return out
}

func writeTempPNG(g image.Image) (string, error) {
	tmp, err := os.CreateTemp("", "ocr-pre-*.png")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	if err := png.Encode(tmp, g); err != nil {
		os.Remove(tmp.Name())
		return "", err
	}
	return tmp.Name(), nil
}

// rankScore menilai hasil OCR untuk voting: NIK + field KTP + panjang.
func rankScore(text string) float64 {
	s := ParseKTP(text)
	score := float64(len(text)) / 1000.0
	score += float64(len(s.FieldConf)) * 2.0
	score += s.Confidence * 5.0
	if s.NIK != "" {
		score += 5
	}
	if s.NIKValid {
		score += 5
	}
	// penalti artefak berat
	_ = sort.StringSlice{}
	return score
}
