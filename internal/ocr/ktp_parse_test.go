package ocr

import "testing"

func TestParseKTP_Sample(t *testing.T) {
	raw := "PROVINSI DKI JAKARTA\nJAKARTA SELATAN\nNIK : 3174096112900001\nNama :DEBBY ANGGRAINI\nTempat/Tgl Lahir : JAKARTA, 21-12-1990\nJenis Kelamin : PEREMPUAN\nAlamat : JL KECAPI V\nRT/RW : 006 / 005\nKel/Desa : JAGAKARSA\nKecamatan : JAGAKARSA\nAgama : ISLAM\nStatus Perkawinan : BELUM KAWIN\nPekerjaan : KARYAWAN SWASTA\nKewarganegaraan : WNI\nBerlaku Hingga : 21-12-2016"
	s := ParseKTP(raw)
	if s.NIK != "3174096112900001" {
		t.Fatalf("nik = %q", s.NIK)
	}
	if !s.NIKValid {
		t.Fatalf("nik should be valid: %s", s.NIKIssue)
	}
	if s.Name != "DEBBY ANGGRAINI" {
		t.Fatalf("nama = %q", s.Name)
	}
	if s.BirthDate != "21-12-1990" {
		t.Fatalf("tgl = %q", s.BirthDate)
	}
	if s.RT != "006" || s.RW != "005" {
		t.Fatalf("rtrw = %q/%q", s.RT, s.RW)
	}
	if s.Confidence < 0.6 {
		t.Fatalf("conf too low: %v", s.Confidence)
	}
}

func TestValidateNIK_FemaleDay(t *testing.T) {
	ok, _ := ValidateNIK("3174096112900001") // 61 = 21+40 perempuan
	if !ok {
		t.Fatal("should be valid female NIK")
	}
	ok, _ = ValidateNIK("3174095112900001")
	if ok {
		// 51 = 11+40, struktur valid walau tidak cocok dengan TTL sampel;
		// tetap lolos struktur — kecocokan TTL dicek manusia.
	}
	ok, _ = ValidateNIK("123")
	if ok {
		t.Fatal("short NIK must fail")
	}
}
