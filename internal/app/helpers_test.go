package app

import (
	"strings"
	"testing"
)

// TestDeedTemplateKeyword memverifikasi pemetaan kode layanan ke kata kunci template.
func TestDeedTemplateKeyword(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"PT-DIRIAN", "PT"},
		{"CV-DIRIAN", "CV"},
		{"PPAT-AJB", "AJB"},
		{"PPAT-HIBAH", "Hibah"},
		{"KUASA", "Kuasa"},
		{"PRENUP", "Kawin"},
		{"WASIAT", "wasiat"},
		{"WASIAT-UMUM", "wasiat"},
		{"PK-UMUM", "Perjanjian"},
		{"LEGALISASI", ""},       // tak ada kata kunci khusus -> fallback kind/kosong
		{"KODE-TAK-KENAL", ""},   // kode tak dikenal -> ""
		{"", ""},                 // kosong aman -> ""
		{"  ppat-ajb  ", "AJB"},  // normalisasi case+spasi
	}
	for _, c := range cases {
		if got := deedTemplateKeyword(c.code); got != c.want {
			t.Errorf("deedTemplateKeyword(%q) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestServiceDeedKind memverifikasi pemetaan kode layanan ke deed kind.
func TestServiceDeedKind(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"PT-DIRIAN", "PT"},
		{"PT-PERUBAHAN", "PT"},
		{"CV-DIRIAN", "CV"},
		{"FIRMA", "Firma"},
		{"YAYASAN-PENDIRIAN", "Yayasan"},
		{"KOPERASI", "Koperasi"},
		{"PPAT-AJB", "AJB"},
		{"PPAT-HIBAH", "Hibah"},
		{"PPAT-TUKAR", "TukarMenukar"},
		{"PPAT-APHB", "APHB"},
		{"WASIAT", "Will"},
		{"WASIAT-UMUM", "Will"},
		{"KUASA", "POA"},
		{"PRENUP", "Prenup"},
		{"HUTANG", "Agreement"},
		{"LEGALISASI", "Other"},
		{"WAARMERKING", "Other"},
		{"AHU-BANTU", "Other"},
		{"KODE-TAK-KENAL", ""}, // default tak dikenal -> ""
		{"", ""},               // kosong aman -> ""
	}
	for _, c := range cases {
		if got := serviceDeedKind(c.code); got != c.want {
			t.Errorf("serviceDeedKind(%q) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestCorpEntityOf memverifikasi pemetaan kode layanan ke tipe badan usaha.
func TestCorpEntityOf(t *testing.T) {
	cases := []struct {
		code string
		want string
	}{
		{"PT-DIRIAN", "PT"},
		{"PT-PERUBAHAN", "PT"},
		{"CV-DIRIAN", "CV"},
		{"FIRMA", "Firma"},
		{"YAYASAN-PENDIRIAN", "Yayasan"},
		{"YAYASAN", "Yayasan"},
		{"KOPERASI", "Koperasi"},
		{"PPAT-AJB", ""},       // non-korporat -> ""
		{"WASIAT", ""},         // non-korporat -> ""
		{"KUASA", ""},          // non-korporat -> ""
		{"KODE-TAK-KENAL", ""}, // tak dikenal -> ""
		{"", ""},               // kosong aman -> ""
	}
	for _, c := range cases {
		if got := corpEntityOf(c.code); got != c.want {
			t.Errorf("corpEntityOf(%q) = %q, want %q", c.code, got, c.want)
		}
	}
}

// TestMaskNIKDisp memverifikasi mask tampilan NIK: hanya 4 digit terakhir terlihat.
func TestMaskNIKDisp(t *testing.T) {
	if got := maskNIKDisp("3201010101010001"); got != "•••• •••••••• 0001" {
		t.Errorf("maskNIKDisp(16 digit) = %q, want mask 4 terakhir", got)
	}
	if got := maskNIKDisp(""); got != "" {
		t.Errorf("maskNIKDisp(\"\") = %q, want \"\"", got)
	}
	if got := maskNIKDisp("   "); got != "" {
		t.Errorf("maskNIKDisp(spasi) = %q, want \"\"", got)
	}
	if got := maskNIKDisp("12"); got != "••••" {
		t.Errorf("maskNIKDisp(pendek) = %q, want \"••••\"", got)
	}
	if got := maskNIKDisp("123"); got != "••••" {
		t.Errorf("maskNIKDisp(3 char) = %q, want \"••••\"", got)
	}
	// Tepat 4 char: tidak bocor lebih dari 4 digit.
	if got := maskNIKDisp("1234"); !strings.HasSuffix(got, "1234") || strings.Contains(got, "1234 ") {
		t.Errorf("maskNIKDisp(4 char) = %q, want suffix 1234", got)
	}
	// Input panjang non-16 digit: tetap hanya 4 terakhir yang tampil.
	if got := maskNIKDisp("AB123456"); !strings.HasSuffix(got, "3456") {
		t.Errorf("maskNIKDisp(panjang) = %q, want suffix 3456", got)
	}
	// Full NIK asli tidak boleh muncul utuh.
	if got := maskNIKDisp("3201010101010001"); strings.Contains(got, "3201010101010001") {
		t.Errorf("maskNIKDisp membocorkan NIK penuh: %q", got)
	}
}

// TestRedactNIKDisp memverifikasi NIK 16 digit pada teks bebas tersamar.
func TestRedactNIKDisp(t *testing.T) {
	in := "NIK 3201010101010001 tercatat"
	got := redactNIKDisp(in)
	if strings.Contains(got, "3201010101010001") {
		t.Errorf("redactNIKDisp tidak menyamarkan NIK: %q", got)
	}
	if !strings.Contains(got, "0001") {
		t.Errorf("redactNIKDisp menghilangkan 4 digit terakhir: %q", got)
	}

	plain := "tidak ada nik di sini"
	if got := redactNIKDisp(plain); got != plain {
		t.Errorf("redactNIKDisp(%q) = %q, want utuh", plain, got)
	}

	if got := redactNIKDisp(""); got != "" {
		t.Errorf("redactNIKDisp(\"\") = %q, want \"\"", got)
	}

	// Digit pendek (<16) tidak boleh diubah.
	short := "kode 12345 selesai"
	if got := redactNIKDisp(short); got != short {
		t.Errorf("redactNIKDisp(%q) = %q, want utuh", short, got)
	}

	// Dua NIK sekaligus: keduanya tersamar.
	multi := "a 3201010101010001 b 3202020202020002 c"
	gotMulti := redactNIKDisp(multi)
	if strings.Contains(gotMulti, "3201010101010001") || strings.Contains(gotMulti, "3202020202020002") {
		t.Errorf("redactNIKDisp tidak menyamarkan semua NIK: %q", gotMulti)
	}
}
