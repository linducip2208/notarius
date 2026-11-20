// Package ocr defines provider-independent OCR/AI interfaces.
// Local-first: external AI is optional and never receives sensitive data
// without authorization. Every result carries provenance and MUST be
// human-verified before it touches identity records or deeds.
package ocr

import (
	"context"
	"errors"
	"strings"
	"time"
)

// DocType classifies an uploaded document.
type DocType string

const (
	DocKTP   DocType = "KTP"
	DocKK    DocType = "KK"
	DocNPWP  DocType = "NPWP"
	DocOther DocType = "OTHER"
)

// Result is one extraction result. VerifiedByHuman is always false
// at creation; a staff member must confirm important fields.
type Result struct {
	ID              string
	DocType         DocType
	Fields          map[string]string
	Confidence      float64
	Provider        string // e.g. "local-tesseract", "manual", "external-ai"
	Provenance      string // binary version / model / config reference
	CreatedAt       time.Time
	VerifiedByHuman bool
	VerifiedBy      *int64
}

// Provider extracts identity fields from a document file.
type Provider interface {
	Name() string
	Extract(ctx context.Context, filePath string) (Result, error)
}

// NoopProvider is used when no OCR engine is installed. It returns a
// descriptive error so the UI can show a manual-entry fallback instead
// of pretending OCR works. See docs/BLOCKERS.md.
type NoopProvider struct{ Reason string }

func (n NoopProvider) Name() string { return "unconfigured" }
func (n NoopProvider) Extract(ctx context.Context, filePath string) (Result, error) {
	return Result{}, errors.New("ocr engine not installed: " + n.Reason)
}

// Classify guesses document type from filename. Advisory only; staff confirm.
func Classify(filename string) DocType {
	l := strings.ToLower(filename)
	switch {
	case strings.Contains(l, "ktp"):
		return DocKTP
	case strings.Contains(l, "kk"), strings.Contains(l, "kartu_keluarga"):
		return DocKK
	case strings.Contains(l, "npwp"):
		return DocNPWP
	}
	return DocOther
}
