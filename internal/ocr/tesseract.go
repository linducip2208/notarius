package ocr

import (
	"context"
	"errors"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// TesseractProvider shells out to a local Tesseract OCR binary (local-first,
// no data leaves the machine). Configure via OCR_BINARY, e.g.
// OCR_BINARY=C:\Program Files\Tesseract-OCR\tesseract.exe
type TesseractProvider struct {
	Binary  string
	Version string
}

// NewTesseract probes the binary; it returns an error when Tesseract is
// missing so the app falls back to NoopProvider with an honest message.
func NewTesseract(binary string) (*TesseractProvider, error) {
	if strings.TrimSpace(binary) == "" {
		return nil, errors.New("ocr: OCR_BINARY tidak diset")
	}
	out, err := exec.Command(binary, "--version").Output()
	if err != nil {
		return nil, err
	}
	ver := strings.TrimSpace(string(out))
	if i := strings.IndexByte(ver, '\n'); i >= 0 {
		ver = ver[:i]
	}
	return &TesseractProvider{Binary: binary, Version: ver}, nil
}

func (t *TesseractProvider) Name() string { return "local-tesseract" }

// Extract runs `tesseract <file> stdout -l ind` and returns raw text in
// Fields["text"] with DocType guessed by Classify. Confidence is left at 0:
// Tesseract's line confidences are unreliable for identity fields, and every
// result MUST be human-verified before touching identity records.
// Extract: enterprise pipeline.
// 1) Quality gate (resolusi/brightness/kontras/blur) — hasilnya disimpan,
//    bukan diblok diam-diam. 2) Preprocessing stdlib (upscale 2x +
//    autocontrast + Otsu) → 3 varian. 3) Voting multi-pass PSM 6/4/3 x
//    lang ind/ind+eng, pemenang = skor kelengkapan field KTP.
// Semua WAJIB verifikasi manusia sebelum menyentuh identitas/akta.
func (t *TesseractProvider) Extract(ctx context.Context, filePath string) (Result, error) {
	q, qerr := AssessImage(filePath)
	fields := map[string]string{}
	if qerr == nil {
		fields["ocr_quality_grade"] = q.Grade
		fields["ocr_quality_score"] = strconv.Itoa(int(q.Score))
		fields["ocr_quality_issues"] = strings.Join(q.Issues, " | ")
		fields["ocr_brightness"] = strconv.FormatFloat(q.Brightness, 'f', 1, 64)
		fields["ocr_contrast"] = strconv.FormatFloat(q.Contrast, 'f', 1, 64)
		fields["ocr_blur"] = strconv.FormatFloat(q.Blur, 'f', 1, 64)
		fields["ocr_resolution"] = strconv.Itoa(q.Width) + "x" + strconv.Itoa(q.Height)
		if q.Reshoot {
			fields["ocr_reshoot"] = "1"
		}
	}
	variants, cleanup, verr := BuildVariants(filePath)
	if verr != nil {
		variants = []Variant{{Path: filePath, Kind: "ORIGINAL"}}
		cleanup = func() {}
	}
	defer cleanup()
	type attempt struct{ psm, lang string }
	passes := []attempt{
		{"6", "ind"}, {"6", "ind+eng"}, {"4", "ind"}, {"3", "ind+eng"},
	}
	best, bestScore := "", -1.0
	bestVariant, bestPSM := "ORIGINAL", "6"
	for _, v := range variants {
		for _, p := range passes {
			cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
			out, err := exec.CommandContext(cmdCtx, t.Binary, v.Path, "stdout",
				"-l", p.lang, "--oem", "1", "--psm", p.psm).Output()
			cancel()
			if err != nil {
				continue
			}
			txt := normalizeOCR(string(out))
			if strings.TrimSpace(txt) == "" {
				continue
			}
			if sc := rankScore(txt); sc > bestScore {
				bestScore, best, bestVariant, bestPSM = sc, txt, v.Kind, p.psm+"/"+p.lang
			}
			if strings.Contains(txt, "NIK") && len(txt) > 400 && rankScore(txt) > 12 {
				break
			}
		}
	}
	if strings.TrimSpace(best) == "" {
		cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		out, err := exec.CommandContext(cmdCtx, t.Binary, filePath, "stdout", "-l", "ind").Output()
		if err != nil {
			return Result{}, err
		}
		best = normalizeOCR(string(out))
	}
	fields["text"] = best
	fields["ocr_variant"] = bestVariant
	fields["ocr_psm"] = bestPSM
	return Result{		DocType:    Classify(filePath),
		Fields:     fields,
		Provider:   t.Name() + "+enterprise-pre",
		Provenance: t.Version,
		CreatedAt:  time.Now().UTC(),
	}, nil
}

// normalizeOCR merapikan spasi & artefak umum tanpa mengubah makna.
func normalizeOCR(s string) string {
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	for i, ln := range lines {
		ln = strings.TrimSpace(ln)
		ln = strings.Join(strings.Fields(ln), " ")
		lines[i] = ln
	}
	s = strings.Join(lines, "\n")
	// "RTIRW." -> "RT/RW", "Tgi" -> "Tgl", pemisah ganda
	reps := map[string]string{"RTIRW": "RT/RW", "Tgi Lahir": "Tgl Lahir", "—": "-", "——": "-"}
	for k, v := range reps {
		s = strings.ReplaceAll(s, k, v)
	}
	return strings.TrimSpace(s)
}
