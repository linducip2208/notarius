package ocr

import (
	"strings"
	"testing"
)

func debbySuggestion(nik string) KTPSuggestion {
	raw := "PROVINSI DKI JAKARTA\nJAKARTA SELATAN\nNIK : " + nik +
		"\nNama :DEBBY ANGGRAINI\nTempat/Tgl Lahir : JAKARTA, 21-12-1990\nJenis Kelamin : PEREMPUAN"
	return ParseKTP(raw)
}

func TestCheckConsistency_ValidFemale(t *testing.T) {
	s := debbySuggestion("3174096112900001") // 61 = 21+40, bulan 12
	if got := CheckConsistency(s); len(got) != 0 {
		t.Fatalf("harusnya lolos, dapat: %v", got)
	}
}

func TestCheckConsistency_DayMismatch(t *testing.T) {
	s := debbySuggestion("3174095112900001") // 51 = 11+40, TTL 21-12-1990
	got := CheckConsistency(s)
	if len(got) == 0 {
		t.Fatal("harusnya flag mismatch hari NIK vs tanggal lahir")
	}
	joined := strings.Join(got, " ")
	if !strings.Contains(strings.ToLower(joined), "tidak cocok") {
		t.Fatalf("pesan harus jelas menyebut ketidakcocokan, dapat: %v", got)
	}
}

func TestCheckConsistency_GenderMismatch(t *testing.T) {
	s := debbySuggestion("3174091112900001") // hari 11 = laki-laki, gender PEREMPUAN
	s.Gender = "PEREMPUAN"
	got := CheckConsistency(s)
	if len(got) == 0 {
		t.Fatal("harusnya flag gender vs NIK")
	}
}

func TestCheckConsistency_MonthMismatch(t *testing.T) {
	s := debbySuggestion("3174096101900001") // bulan NIK 01 vs lahir 12
	got := CheckConsistency(s)
	if len(got) == 0 {
		t.Fatal("harusnya flag bulan NIK vs bulan lahir")
	}
}

func TestCheckConsistency_SamarTolerated(t *testing.T) {
	s := debbySuggestion("3174096112900001")
	s.BirthDate = "M-2-1990" // samar: jangan flag hari/bulan
	if got := CheckConsistency(s); len(got) != 0 {
		t.Fatalf("tanggal samar harus ditoleransi, dapat: %v", got)
	}
	s.BirthDate = ""
	if got := CheckConsistency(s); len(got) != 0 {
		t.Fatalf("tanggal kosong harus ditoleransi, dapat: %v", got)
	}
}
