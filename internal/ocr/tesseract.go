package ocr

import (
	"context"
	"errors"
	"os/exec"
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
func (t *TesseractProvider) Extract(ctx context.Context, filePath string) (Result, error) {
	cmdCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	out, err := exec.CommandContext(cmdCtx, t.Binary, filePath, "stdout", "-l", "ind").Output()
	if err != nil {
		return Result{}, err
	}
	return Result{
		DocType:   Classify(filePath),
		Fields:    map[string]string{"text": string(out)},
		Provider:  t.Name(),
		Provenance: t.Version,
		CreatedAt: time.Now().UTC(),
	}, nil
}
