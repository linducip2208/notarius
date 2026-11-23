package ocr

import (
	"regexp"
	"strconv"
	"strings"
)

var reBirthDateStrict = regexp.MustCompile(`\b(\d{1,2})[\-\/](\d{1,2})[\-\/](\d{2,4})\b`)
var reDigitsOnly = regexp.MustCompile(`\D`)

// CheckConsistency memeriksa silang field hasil ParseKTP.
// Hanya saran review (tidak pernah auto-verify): kembalikan daftar masalah
// berbahasa Indonesia; kosong bila konsisten atau data pembanding samar/absen.
func CheckConsistency(s KTPSuggestion) []string {
	var out []string

	d := reDigitsOnly.ReplaceAllString(s.NIK, "")
	if len(d) != 16 {
		return nil // NIK tak terbaca / samar: sudah di-flag ValidateNIK
	}
	for _, c := range d {
		if c < '0' || c > '9' {
			return nil
		}
	}
	dayCode, _ := strconv.Atoi(d[6:8])
	monthCode, _ := strconv.Atoi(d[8:10])

	// --- NIK vs tanggal/bulan lahir ---
	bd := strings.TrimSpace(s.BirthDate)
	if m := reBirthDateStrict.FindStringSubmatch(bd); m != nil {
		dd, _ := strconv.Atoi(m[1])
		mm, _ := strconv.Atoi(m[2])
		// Hari: cocok bila dayCode == DD (laki-laki) atau DD+40 (perempuan).
		matchDay := dayCode == dd || dayCode == dd+40
		validDayRange := (dayCode >= 1 && dayCode <= 31) || (dayCode >= 41 && dayCode <= 71)
		if !validDayRange {
			out = append(out, "kode hari NIK ("+d[6:8]+") tidak valid — cek digit NIK per digit dengan fisik KTP")
		} else if !matchDay {
			out = append(out, "hari NIK ("+d[6:8]+") tidak cocok dengan tanggal lahir "+bd+
				" (diharapkan "+two(dd)+" untuk LAKI-LAKI atau "+two(dd+40)+" untuk PEREMPUAN) — cek fisik KTP")
		}
		// Bulan: kode bulan NIK vs MM tanggal lahir.
		if monthCode >= 1 && monthCode <= 12 && mm >= 1 && mm <= 12 {
			if monthCode != mm {
				out = append(out, "bulan NIK ("+d[8:10]+") tidak cocok dengan bulan lahir "+bd+
					" — cek fisik KTP")
			}
		}
	}
	// Bila BirthDate samar (mis. "M-2-1990") atau kosong: toleransi, tanpa flag hari/bulan.

	// --- Gender vs NIK ---
	g := strings.ToUpper(strings.TrimSpace(s.Gender))
	if g != "" {
		isFemale := strings.Contains(g, "PEREMPUAN")
		isMale := strings.Contains(g, "LAKI")
		switch {
		case dayCode >= 41 && dayCode <= 71:
			if !isFemale {
				out = append(out, "NIK menandakan PEREMPUAN (kode hari "+d[6:8]+
					" = tanggal+40) tetapi jenis kelamin terbaca \""+s.Gender+"\" — cek fisik KTP")
			}
		case dayCode >= 1 && dayCode <= 31:
			if !isMale {
				out = append(out, "NIK menandakan LAKI-LAKI (kode hari "+d[6:8]+
					") tetapi jenis kelamin terbaca \""+s.Gender+"\" — cek fisik KTP")
			}
		}
	}

	return out
}

func two(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}
