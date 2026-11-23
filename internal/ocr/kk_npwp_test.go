package ocr

import (
	"strings"
	"testing"
)

const sampleKK = `KARTU KELUARGA
No. KK : 3174090101200001
Kepala Keluarga : BUDI SANTOSO
Alamat : JL MELATI I NO 10
RT/RW : 003 / 007
Kel/Desa : JAGAKARSA
Kecamatan : JAGAKARSA
1. BUDI SANTOSO 3174090101800001
2. SARI WULANDARI 3174095201850002
3. ANDI PRATAMA 3174091505100003`

const sampleNPWP = `KARTU NOMOR POKOK WAJIB PAJAK
NPWP : 09.123.456.7-123.456
Nama : BUDI SANTOSO
Alamat : JL MELATI I NO 10 JAKARTA`

func TestParseKK_Sample(t *testing.T) {
	k := ParseKK(sampleKK)
	if k.NoKK != "3174090101200001" {
		t.Fatalf("no kk = %q", k.NoKK)
	}
	if !k.NoKKValid {
		t.Fatalf("no kk harus valid: %s", k.NoKKIssue)
	}
	if k.HeadName != "BUDI SANTOSO" {
		t.Fatalf("kepala = %q", k.HeadName)
	}
	if len(k.Members) != 3 {
		t.Fatalf("anggota = %d, want 3", len(k.Members))
	}
	if k.Members[0].NIK != "3174090101800001" || k.Members[0].Name != "BUDI SANTOSO" {
		t.Fatalf("anggota1 = %+v", k.Members[0])
	}
	m := k.ToMap()
	for _, key := range []string{"kk_no", "kk_kepala", "kk_alamat", "kk_anggota", "kk_jumlah", "kk_nama_1", "kk_nik_1"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("ToMap kurang kunci %s", key)
		}
	}
	if m["kk_jumlah"] != "3" {
		t.Fatalf("kk_jumlah = %q", m["kk_jumlah"])
	}
	if !strings.Contains(m["kk_anggota"], "SARI WULANDARI") {
		t.Fatalf("kk_anggota = %q", m["kk_anggota"])
	}
}

func TestParseKK_MemberCap(t *testing.T) {
	var b strings.Builder
	b.WriteString("KARTU KELUARGA\nNo. KK : 3174090101200001\n")
	for i := 0; i < 12; i++ {
		b.WriteString("ANGGOTA KELUARGA 31740901018000" + string(rune('0'+i%10)) + "\n")
	}
	k := ParseKK(b.String())
	if len(k.Members) > 10 {
		t.Fatalf("anggota harus dibatasi 10, dapat %d", len(k.Members))
	}
}

func TestParseNPWP_Sample(t *testing.T) {
	n := ParseNPWP(sampleNPWP)
	if n.NPWP != "091234567123456" {
		t.Fatalf("npwp = %q", n.NPWP)
	}
	if !n.NPWPValid {
		t.Fatalf("npwp harus valid: %s", n.NPWPIssue)
	}
	if n.Formatted != "09.123.456.7-123.456" {
		t.Fatalf("formatted = %q", n.Formatted)
	}
	if n.Name != "BUDI SANTOSO" {
		t.Fatalf("nama = %q", n.Name)
	}
	m := n.ToMap()
	for _, key := range []string{"npwp_no", "npwp_formatted", "npwp_valid", "npwp_nama", "npwp_alamat"} {
		if _, ok := m[key]; !ok {
			t.Fatalf("ToMap kurang kunci %s", key)
		}
	}
}

func TestValidateNPWP_Formats(t *testing.T) {
	if ok, _ := ValidateNPWP("09.123.456.7-123.456"); !ok {
		t.Fatal("15 digit berformat harus valid")
	}
	if ok, _ := ValidateNPWP("3174096112900001"); !ok {
		t.Fatal("16 digit harus valid (format baru)")
	}
	if ok, _ := ValidateNPWP("123"); ok {
		t.Fatal("pendek harus gagal")
	}
}
