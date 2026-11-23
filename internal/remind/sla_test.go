package remind

import (
	"testing"
)

func TestServiceSLADefaults(t *testing.T) {
	cases := []struct {
		code string
		want int
	}{
		{"LEGALISASI", 2},
		{"waarmerking-xyz", 2},
		{"PPAT-AJB", 7},
		{"SKMHT", 7},
		{"APHT-01", 7},
		{"PT-Pendirian", 14},
		{"CV-Berkat", 14},
		{"FIRMA", 14},
		{"YAYASAN-Pendidikan", 14},
		{"KOPERASI-Maju", 14},
		{"WASIAT-Olograf", 3},
		{"AHU-Online", 10},
		{"OSS-RBA", 10},
		{"LAINNYA", 7},
		{"", 7},
	}
	for _, c := range cases {
		if got := ServiceSLA(c.code); got != c.want {
			t.Errorf("ServiceSLA(%q) = %d, want %d", c.code, got, c.want)
		}
	}
}

func TestParseSLAOverridesValid(t *testing.T) {
	m, err := ParseSLAOverrides(`{"PPAT-AJB": 5, "default": 10}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["PPAT-AJB"] != 5 {
		t.Errorf("PPAT-AJB = %d, want 5", m["PPAT-AJB"])
	}
	if m["DEFAULT"] != 10 {
		t.Errorf("DEFAULT = %d, want 10", m["DEFAULT"])
	}

	// Case-insensitive keys dinormalisasi ke huruf besar.
	m, err = ParseSLAOverrides(`{"ppat-ajb": 3}`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if m["PPAT-AJB"] != 3 {
		t.Errorf("PPAT-AJB = %d, want 3", m["PPAT-AJB"])
	}

	// Kosong/NULL = tidak ada override, tanpa error.
	for _, raw := range []string{"", "   ", "null"} {
		m, err = ParseSLAOverrides(raw)
		if err != nil {
			t.Errorf("ParseSLAOverrides(%q) error = %v, want nil", raw, err)
		}
		if len(m) != 0 {
			t.Errorf("ParseSLAOverrides(%q) = %v, want empty", raw, m)
		}
	}
}

func TestParseSLAOverridesInvalid(t *testing.T) {
	for _, raw := range []string{
		`{broken`,
		`[1,2]`,
		`"PPAT-AJB"`,
		`{"PPAT-AJB": 0}`,
		`{"PPAT-AJB": -2}`,
		`{"PPAT-AJB": 400}`,
		`{"PPAT-AJB": "lima"}`,
		`{"PPAT-AJB": 2.5}`,
		`{"": 5}`,
	} {
		if _, err := ParseSLAOverrides(raw); err == nil {
			t.Errorf("ParseSLAOverrides(%q) = nil error, want error", raw)
		}
	}
}

func TestResolveSLA(t *testing.T) {
	over := map[string]int{"PPAT-AJB": 5, "DEFAULT": 10}
	if got := ResolveSLA("PPAT-AJB", over); got != 5 {
		t.Errorf("exact override = %d, want 5", got)
	}
	if got := ResolveSLA("ppat-ajb", over); got != 5 {
		t.Errorf("case-insensitive override = %d, want 5", got)
	}
	if got := ResolveSLA("WASIAT", over); got != 10 {
		t.Errorf("default override = %d, want 10", got)
	}
	// Tanpa override / map kosong / JSON rusak (nil) -> default lama.
	if got := ResolveSLA("PPAT-AJB", nil); got != ServiceSLA("PPAT-AJB") {
		t.Errorf("nil map = %d, want ServiceSLA %d", got, ServiceSLA("PPAT-AJB"))
	}
	if got := ResolveSLA("WASIAT", map[string]int{}); got != 3 {
		t.Errorf("empty map = %d, want 3", got)
	}
	broken, err := ParseSLAOverrides(`{oops`)
	if err == nil {
		t.Fatal("expected parse error for broken JSON")
	}
	_ = broken // nil; pastikan fallback dipakai pemanggil
	if got := ResolveSLA("LEGALISASI", broken); got != 2 {
		t.Errorf("broken JSON fallback = %d, want 2", got)
	}
}

func TestServiceSLAForFailOpen(t *testing.T) {
	// DB nil (atau tabel hilang di loadSLAOverrides) -> fallback default lama.
	if got := ServiceSLAFor(nil, 1, "PPAT-AJB"); got != ServiceSLA("PPAT-AJB") {
		t.Errorf("ServiceSLAFor(nil) = %d, want %d", got, ServiceSLA("PPAT-AJB"))
	}
	if got := ServiceSLAFor(nil, 1, "WASIAT"); got != 3 {
		t.Errorf("ServiceSLAFor(nil) = %d, want 3", got)
	}
}
