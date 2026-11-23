package app

import (
	"strings"
	"testing"
)

// TestStandardChecklistSeeds_NewCodes memastikan seed checklist memuat
// HUTANG (5 item), PK-UMUM (4 item), SEWA (4 item) — murni tanpa DB.
func TestStandardChecklistSeeds_NewCodes(t *testing.T) {
	seeds := standardChecklistSeeds()
	byCode := map[string]checklistSeed{}
	for _, s := range seeds {
		if _, dup := byCode[s.code]; dup {
			t.Errorf("duplikat service_code %q", s.code)
		}
		byCode[s.code] = s
	}
	want := map[string]int{"HUTANG": 5, "PK-UMUM": 4, "SEWA": 4}
	for code, n := range want {
		s, ok := byCode[code]
		if !ok {
			t.Errorf("seed checklist %q tidak ditemukan", code)
			continue
		}
		if len(s.items) != n {
			t.Errorf("seed %q = %d item, want %d", code, len(s.items), n)
		}
		if strings.TrimSpace(s.name) == "" {
			t.Errorf("seed %q nama kosong", code)
		}
	}
	// Isi kunci sesuai spesifikasi (pola sama seperti entri existing).
	h := byCode["HUTANG"]
	joined := strings.ToLower(strings.Join(h.items, " | "))
	for _, kw := range []string{"ktp", "pokok", "angsuran", "jaminan", "kuasa"} {
		if !strings.Contains(joined, kw) {
			t.Errorf("seed HUTANG tidak memuat kata kunci %q: %q", kw, joined)
		}
	}
	sw := byCode["SEWA"]
	joined = strings.ToLower(strings.Join(sw.items, " | "))
	for _, kw := range []string{"ktp", "objek", "jangka waktu", "harga"} {
		if !strings.Contains(joined, kw) {
			t.Errorf("seed SEWA tidak memuat kata kunci %q: %q", kw, joined)
		}
	}
}

// TestStandardTemplateSeeds_Wasiat memastikan template konsep Wasiat ada
// (kategori Will, variabel {{client_name}}, header KONSEP, akses terbatas,
// saksi) sehingga keyword "wasiat" autoDeedTemplate menemukan fallback DRAFT.
func TestStandardTemplateSeeds_Wasiat(t *testing.T) {
	var found *templateSeed
	for _, s := range standardTemplateSeeds() {
		if s.name == "Standar Wasiat" {
			c := s
			found = &c
		}
	}
	if found == nil {
		t.Fatal("template konsep \"Standar Wasiat\" tidak ditemukan")
	}
	if found.cat != "Will" {
		t.Errorf("kategori Wasiat = %q, want %q", found.cat, "Will")
	}
	for _, kw := range []string{"KONSEP", "{{client_name}}", "saksi", "terbatas"} {
		if !strings.Contains(found.body, kw) {
			t.Errorf("body Wasiat tidak memuat %q", kw)
		}
	}
	// Keyword autoDeedTemplate untuk WASIAT harus cocok via name/category.
	if got := deedTemplateKeyword("WASIAT"); !strings.Contains(strings.ToLower("Standar Wasiat Will "+found.body), strings.ToLower(got)) && got != "wasiat" {
		t.Errorf("deedTemplateKeyword(WASIAT) = %q, want wasiat agar fallback cocok", got)
	}
}
