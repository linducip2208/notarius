package wib

import (
	"testing"
	"time"
)

func TestFormatNonZero(t *testing.T) {
	s := Format(time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC))
	if s == "-" || s == "" {
		t.Fatal("format kosong")
	}
	if got := Date(time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)); got == "-" {
		t.Fatal("date kosong")
	}
}
