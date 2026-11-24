package dms

import "testing"

func sugCats() []Category {
	return []Category{
		{ID: 1, Name: "Identitas"},
		{ID: 2, Name: "Pajak"},
		{ID: 3, Name: "Akta"},
		{ID: 4, Name: "Keluarga"},
	}
}

func TestSuggestKTP(t *testing.T) {
	hit, ok := SuggestCategory(sugCats(), "KTP", "image/jpeg", "ktp-ibu.jpg")
	if !ok || hit.Name != "Identitas" || hit.ID != 1 {
		t.Fatalf("KTP should suggest Identitas, got %+v %v", hit, ok)
	}
}

func TestSuggestKKPrefersKeluarga(t *testing.T) {
	hit, ok := SuggestCategory(sugCats(), "KK", "image/jpeg", "kk.jpg")
	if !ok || hit.Name != "Keluarga" {
		t.Fatalf("KK should prefer Keluarga, got %+v %v", hit, ok)
	}
}

func TestSuggestKKFallsBackToIdentitas(t *testing.T) {
	cats := []Category{{ID: 7, Name: "identitas"}} // no Keluarga configured
	hit, ok := SuggestCategory(cats, "KK", "image/jpeg", "kk.jpg")
	if !ok || hit.ID != 7 {
		t.Fatalf("KK should fall back to Identitas, got %+v %v", hit, ok)
	}
}

func TestSuggestNPWPCaseInsensitive(t *testing.T) {
	cats := []Category{{ID: 9, Name: "  PAJAK "}} // spacing + case must still match
	hit, ok := SuggestCategory(cats, "npwp", "application/pdf", "npwp.pdf")
	if !ok || hit.ID != 9 {
		t.Fatalf("NPWP should match PAJAK case-insensitively, got %+v %v", hit, ok)
	}
}

func TestSuggestPDFAkta(t *testing.T) {
	hit, ok := SuggestCategory(sugCats(), "OTHER", "application/pdf", "Akta-Jual-Beli-12.pdf")
	if !ok || hit.Name != "Akta" {
		t.Fatalf("PDF akta should suggest Akta, got %+v %v", hit, ok)
	}
}

func TestSuggestNoMatchReturnsFalse(t *testing.T) {
	if _, ok := SuggestCategory(sugCats(), "OTHER", "image/png", "foto-acara.png"); ok {
		t.Fatal("unrelated OTHER must not suggest anything")
	}
	if _, ok := SuggestCategory(sugCats(), "KTP", "image/jpeg", "ktp.jpg"); !ok {
		t.Fatal("sanity: KTP with full catalog must match")
	}
	// Catalog without Identitas: KTP has nowhere to go — and must NOT create one.
	if _, ok := SuggestCategory([]Category{{ID: 5, Name: "Keuangan"}}, "KTP", "image/jpeg", "ktp.jpg"); ok {
		t.Fatal("missing category must yield ok=false (never auto-create)")
	}
}

func TestSuggestEmptyCatalog(t *testing.T) {
	if _, ok := SuggestCategory(nil, "KK", "image/jpeg", "kk.jpg"); ok {
		t.Fatal("empty catalog must yield ok=false")
	}
}

func TestSuggestDoesNotCreate(t *testing.T) {
	// Purity: input slice must be unchanged (no appends, no renames).
	cats := sugCats()
	n := len(cats)
	before := make([]Category, n)
	copy(before, cats)
	_, _ = SuggestCategory(cats, "KTP", "image/jpeg", "ktp.jpg")
	_, _ = SuggestCategory(cats, "OTHER", "application/pdf", "akta-x.pdf")
	if len(cats) != n {
		t.Fatal("SuggestCategory must not append categories")
	}
	for i := range cats {
		if cats[i] != before[i] {
			t.Fatalf("SuggestCategory must not mutate catalog entry %d", i)
		}
	}
}
