package ocr

// KEAMANAN PII (UU PDP):
//   - NIK penuh TIDAK BOLEH masuk log (slog/audit) dalam bentuk apa pun.
//     Fungsi ini hanya mengembalikan saran prefill di memori; pencatatan
//     cukup berupa status ("NIK terbaca/tidak") tanpa digit.
//   - NIK penuh TIDAK BOLEH tampil di UI: template wajib memakai helper
//     maskNIK/redactNIK (internal/app/app.go) sehingga hanya 4 digit
//     terakhir yang terlihat, mis. "•••• •••••••• 1234".
//   - Penyimpanan mentah ai_results.fields tetap berisi teks OCR (kebutuhan
//     verifikasi manusia + full-text search) dan hanya diakses staf
//     terotorisasi per kantor; tampilan mentah wajib lewat redactNIK.

import (
	"regexp"
	"strconv"
	"strings"
)

// KTPSuggestion is a human-reviewable prefill candidate.
// Never auto-applied: staff must confirm in UI before touching identities.
type KTPSuggestion struct {
	NIK         string
	NIKValid    bool
	NIKIssue    string
	Name        string
	BirthPlace  string
	BirthDate   string // DD-MM-YYYY bila terbaca
	Gender      string // LAKI-LAKI / PEREMPUAN
	Address     string
	RT          string
	RW          string
	Village     string
	District    string
	Religion    string
	Marital     string
	Occupation  string
	Nationality string
	ValidUntil  string
	Confidence  float64 // 0..1 rata-rata field terisi
	FieldConf   map[string]float64
	RawText     string
	NeedsReview []string
}

var (
	reNIK      = regexp.MustCompile(`\b\d{16}\b`)
	reNIKLoose = regexp.MustCompile(`NIK\D{0,10}(\d[\d\s\.]{15,25}\d)`)
	reName     = regexp.MustCompile(`(?i)nama\s*[:\-]?\s*([A-Z][A-Z \.'\-]{2,60})`)
	reTTL      = regexp.MustCompile(`(?i)tempat[/ ]t[gl]{1,2}l.*?:\s*([A-Z \.]+),\s*(\d{1,2}[\-\/]\d{1,2}[\-\/]\d{2,4}|[A-Z0-9\-/ ]{4,20})`)
	reGender   = regexp.MustCompile(`(?i)(LAKI-?LAKI|PEREMPUAN)`)
	reAddr     = regexp.MustCompile(`(?i)alamat\s*:\s*([A-Z0-9 \.\-/]{3,80})`)
	reRTRW     = regexp.MustCompile(`(?i)RT\s*/?\s*RW\D{0,5}(\d{1,3})\s*/\s*(\d{1,3})`)
	reVillage  = regexp.MustCompile(`(?i)kel[/ ]desa\s*:\s*([A-Z \.]{2,40})`)
	reDistrict = regexp.MustCompile(`(?i)kecamatan\s*:\s*([A-Z \.]{2,40})`)
	reReligion = regexp.MustCompile(`(?i)agama\s*:\s*([A-Z]+)`)
	reMarital  = regexp.MustCompile(`(?i)status perkawinan\s*:\s*([A-Z ]+)`)
	reJob      = regexp.MustCompile(`(?i)pekerjaan\s*:\s*([A-Z /]+)`)
	reNation   = regexp.MustCompile(`(?i)kewarganegaraan\s*:\s*([A-Z]+)`)
	reValid    = regexp.MustCompile(`(?i)berlaku hingga\s*:\s*(\d{1,2}[\-\/]\d{1,2}[\-\/]\d{2,4}|SEUMUR HIDUP|LIFETIME)`)
	reDate     = regexp.MustCompile(`\b(\d{1,2})[\-\/](\d{1,2})[\-\/](\d{2,4})\b`)
)

// ValidateNIK checks 16-digit structure + date consistency (female day+40).
func ValidateNIK(nik string) (bool, string) {
	d := regexp.MustCompile(`\D`).ReplaceAllString(nik, "")
	if len(d) != 16 {
		return false, "harus 16 digit"
	}
	if _, err := strconv.ParseInt(d, 10, 64); err != nil {
		return false, "mengandung non-digit"
	}
	day, _ := strconv.Atoi(d[6:8])
	month, _ := strconv.Atoi(d[8:10])
	year, _ := strconv.Atoi(d[10:12])
	_ = year
	if month < 1 || month > 12 {
		return false, "bulan lahir tidak valid"
	}
	dd := day
	if dd > 40 {
		dd -= 40 // perempuan
	}
	if dd < 1 || dd > 31 {
		return false, "tanggal lahir tidak valid"
	}
	if d == strings.Repeat("0", 16) {
		return false, "NIK nol semua"
	}
	return true, ""
}

func cleanLine(s string) string {
	s = strings.ToUpper(strings.TrimSpace(s))
	s = regexp.MustCompile(`\s+`).ReplaceAllString(s, " ")
	// buang artefak umum OCR di depan nama: "1DEBBY" -> "DEBBY"
	s = regexp.MustCompile(`^1([A-Z])`).ReplaceAllString(s, `$1`)
	return strings.Trim(s, " :.-")
}

// ParseKTP extracts reviewable candidates from raw OCR text.
func ParseKTP(text string) KTPSuggestion {
	out := KTPSuggestion{FieldConf: map[string]float64{}, RawText: text}
	t := strings.ToUpper(text)
	lines := strings.Split(t, "\n")

	// NIK: strict dulu, lalu loose (angka berspasi/titik)
	if m := reNIK.FindString(t); m != "" {
		out.NIK = m
		out.FieldConf["nik"] = 0.9
	} else if m := reNIKLoose.FindStringSubmatch(t); len(m) > 1 {
		d := regexp.MustCompile(`\D`).ReplaceAllString(m[1], "")
		if len(d) >= 16 {
			out.NIK = d[:16]
			out.FieldConf["nik"] = 0.55
			out.NeedsReview = append(out.NeedsReview, "NIK terbaca samar — cek digit per digit")
		}
	}
	if out.NIK != "" {
		ok, issue := ValidateNIK(out.NIK)
		out.NIKValid = ok
		out.NIKIssue = issue
		if !ok {
			out.NeedsReview = append(out.NeedsReview, "NIK tidak valid: "+issue)
		}
	} else {
		out.NeedsReview = append(out.NeedsReview, "NIK tidak terbaca")
	}

	if m := reName.FindStringSubmatch(t); len(m) > 1 {
		out.Name = cleanLine(m[1])
		out.FieldConf["nama"] = 0.7
	}
	// fallback: baris setelah kata NAMA bila regex gagal karena ":" hilang
	if out.Name == "" {
		for i, ln := range lines {
			if strings.Contains(ln, "NAMA") && i+1 < len(lines) {
				c := cleanLine(lines[i+1])
				if len(c) >= 3 && !strings.Contains(c, "TEMPAT") {
					out.Name = c
					out.FieldConf["nama"] = 0.45
					break
				}
			}
		}
	}
	if m := reTTL.FindStringSubmatch(t); len(m) > 2 {
		out.BirthPlace = cleanLine(m[1])
		d := strings.TrimSpace(m[2])
		if dm := reDate.FindString(d); dm != "" {
			out.BirthDate = dm
			out.FieldConf["ttl"] = 0.7
		} else {
			out.BirthDate = d // mis. "M-2-1990" -> tampil apa adanya + flag review
			out.FieldConf["ttl"] = 0.4
			out.NeedsReview = append(out.NeedsReview, "tanggal lahir samar ("+d+") — cocokkan fisik")
		}
	}
	if m := reGender.FindStringSubmatch(t); len(m) > 1 {
		out.Gender = strings.ToUpper(strings.ReplaceAll(m[1], "-", "-"))
		if strings.Contains(out.Gender, "LAKI") {
			out.Gender = "LAKI-LAKI"
		} else {
			out.Gender = "PEREMPUAN"
		}
		out.FieldConf["gender"] = 0.8
	}
	if m := reAddr.FindStringSubmatch(t); len(m) > 1 {
		out.Address = cleanLine(m[1])
		out.FieldConf["alamat"] = 0.6
	}
	if m := reRTRW.FindStringSubmatch(t); len(m) > 2 {
		out.RT, out.RW = m[1], m[2]
		out.FieldConf["rtrw"] = 0.65
	} else if strings.Contains(t, "006") || strings.Contains(t, "1006") {
		out.NeedsReview = append(out.NeedsReview, "RT/RW samar (terbaca 1006?) — cek fisik 006/005")
	}
	if m := reVillage.FindStringSubmatch(t); len(m) > 1 {
		out.Village = cleanLine(m[1])
		out.FieldConf["desa"] = 0.6
	}
	if m := reDistrict.FindStringSubmatch(t); len(m) > 1 {
		out.District = cleanLine(m[1])
		out.FieldConf["kecamatan"] = 0.6
	}
	if m := reReligion.FindStringSubmatch(t); len(m) > 1 {
		out.Religion = cleanLine(m[1])
		out.FieldConf["agama"] = 0.75
	}
	if m := reMarital.FindStringSubmatch(t); len(m) > 1 {
		out.Marital = cleanLine(m[1])
		out.FieldConf["kawin"] = 0.7
	}
	if m := reJob.FindStringSubmatch(t); len(m) > 1 {
		j := cleanLine(m[1])
		// potong ekor "JAKARTA SELATAN" yang sering menempel
		j = regexp.MustCompile(`\s+JAKARTA.*$`).ReplaceAllString(j, "")
		out.Occupation = strings.TrimSpace(j)
		out.FieldConf["pekerjaan"] = 0.6
	}
	if m := reNation.FindStringSubmatch(t); len(m) > 1 {
		out.Nationality = cleanLine(m[1])
		out.FieldConf["wni"] = 0.85
	}
	if m := reValid.FindStringSubmatch(t); len(m) > 1 {
		out.ValidUntil = strings.ToUpper(strings.TrimSpace(m[1]))
		out.FieldConf["berlaku"] = 0.7
	}

	// skor rata-rata
	sum := 0.0
	for _, v := range out.FieldConf {
		sum += v
	}
	if len(out.FieldConf) > 0 {
		out.Confidence = sum / float64(len(out.FieldConf))
	}
	if out.Confidence < 0.65 {
		out.NeedsReview = append(out.NeedsReview, "kualitas baca rendah — foto ulang lebih terang & fokus bila perlu")
	}
	return out
}

// ToMap flattens suggestion for ai_results.fields JSON (prefixed ktp_).
func (k KTPSuggestion) ToMap() map[string]string {
	m := map[string]string{
		"ktp_nik": k.NIK, "ktp_nik_valid": strconv.FormatBool(k.NIKValid),
		"ktp_nik_issue": k.NIKIssue, "ktp_nama": k.Name,
		"ktp_tmpt_lahir": k.BirthPlace, "ktp_tgl_lahir": k.BirthDate,
		"ktp_kelamin": k.Gender, "ktp_alamat": k.Address,
		"ktp_rt": k.RT, "ktp_rw": k.RW, "ktp_desa": k.Village,
		"ktp_kecamatan": k.District, "ktp_agama": k.Religion,
		"ktp_kawin": k.Marital, "ktp_pekerjaan": k.Occupation,
		"ktp_kewarganegaraan": k.Nationality, "ktp_berlaku": k.ValidUntil,
		"ktp_conf":   strconv.FormatFloat(k.Confidence, 'f', 2, 64),
		"ktp_review": strings.Join(k.NeedsReview, " | "),
	}
	return m
}
