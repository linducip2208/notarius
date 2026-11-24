package ocr

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// --- KK ---

// KKMember adalah satu anggota keluarga (maksimal 10 baris disimpan).
type KKMember struct {
	Name string
	NIK  string
}

// KKSuggestion adalah kandidat prefill Kartu Keluarga (review-only).
type KKSuggestion struct {
	NoKK        string
	NoKKValid   bool
	NoKKIssue   string
	HeadName    string // nama kepala keluarga
	Address     string
	RT          string
	RW          string
	Village     string
	District    string
	Members     []KKMember
	Confidence  float64
	NeedsReview []string
	RawText     string
}

var (
	reKKLabel = regexp.MustCompile(`(?i)(?:NO\.?\s*(?:KK|KARTU KELUARGA)|NOMOR\s*(?:KK|KARTU KELUARGA)|KK\s*(?:NO|NOMOR)|KARTU KELUARGA)[\s:.\-]*([\d][\d\s\.\-]{14,30}\d)`)
	reKKHead  = regexp.MustCompile(`(?i)KEPALA\s*KELUARGA\s*[:\-]?\s*([A-Z][A-Z \.'\-]{2,60})`)
	reKKAddr  = regexp.MustCompile(`(?i)ALAMAT\s*[:\-]?\s*([A-Z0-9 \.\-/]{3,80})`)
	reKKName  = regexp.MustCompile(`[A-Z]{3,}(?:[ \.'][A-Z\-]+)+`)
	reKKLeadN = regexp.MustCompile(`^\s*\d{1,2}[\.\)]\s*`)
)

// ValidateKKNo memeriksa struktur 16 digit nomor KK.
func ValidateKKNo(no string) (bool, string) {
	d := reDigitsOnly.ReplaceAllString(no, "")
	if len(d) != 16 {
		return false, "harus 16 digit"
	}
	if _, err := strconv.ParseInt(d, 10, 64); err != nil {
		return false, "mengandung non-digit"
	}
	if d == strings.Repeat("0", 16) {
		return false, "nomor nol semua"
	}
	return true, ""
}

// ParseKK mengekstrak saran prefill dari teks OCR Kartu Keluarga.
func ParseKK(text string) KKSuggestion {
	out := KKSuggestion{RawText: text}
	t := strings.ToUpper(text)

	// Nomor KK: berlabel dulu, lalu fallback digit 16 pertama bila konteks KK jelas.
	if m := reKKLabel.FindStringSubmatch(t); len(m) > 1 {
		d := reDigitsOnly.ReplaceAllString(m[1], "")
		if len(d) >= 16 {
			out.NoKK = d[:16]
		}
	}
	if out.NoKK == "" && strings.Contains(t, "KARTU KELUARGA") {
		if m := reNIK.FindString(t); m != "" {
			out.NoKK = m
			out.NeedsReview = append(out.NeedsReview, "nomor KK tanpa label jelas — pastikan bukan NIK anggota")
		}
	}
	if out.NoKK != "" {
		ok, issue := ValidateKKNo(out.NoKK)
		out.NoKKValid = ok
		out.NoKKIssue = issue
		if !ok {
			out.NeedsReview = append(out.NeedsReview, "nomor KK tidak valid: "+issue)
		}
	} else {
		out.NeedsReview = append(out.NeedsReview, "nomor KK tidak terbaca")
	}

	if m := reKKHead.FindStringSubmatch(t); len(m) > 1 {
		out.HeadName = cleanLine(m[1])
	} else {
		lines := strings.Split(t, "\n")
		for i, ln := range lines {
			if strings.Contains(ln, "KEPALA KELUARGA") && i+1 < len(lines) {
				if c := cleanLine(lines[i+1]); len(c) >= 3 {
					out.HeadName = c
					break
				}
			}
		}
	}
	if m := reKKAddr.FindStringSubmatch(t); len(m) > 1 {
		out.Address = cleanLine(m[1])
	}
	if m := reRTRW.FindStringSubmatch(t); len(m) > 2 {
		out.RT, out.RW = m[1], m[2]
	}
	if m := reVillage.FindStringSubmatch(t); len(m) > 1 {
		out.Village = cleanLine(m[1])
	}
	if m := reDistrict.FindStringSubmatch(t); len(m) > 1 {
		out.District = cleanLine(m[1])
	}

	// Anggota: tiap baris ber-NIK 16 digit → 1 anggota (maks 10).
	for _, ln := range strings.Split(t, "\n") {
		if len(out.Members) >= 10 {
			break
		}
		nik := reNIK.FindString(ln)
		if nik == "" {
			continue
		}
		// Jangan jadikan nomor KK itu sendiri sebagai anggota bila label No.KK satu baris.
		if nik == out.NoKK && strings.Contains(ln, "KK") {
			continue
		}
		rest := strings.Replace(ln, nik, " ", 1)
		rest = reKKLeadN.ReplaceAllString(rest, " ")
		rest = regexp.MustCompile(`\bNIK\b|\bNAMA\b|\bNO\b`).ReplaceAllString(rest, " ")
		name := ""
		if cands := reKKName.FindAllString(cleanLine(rest), -1); len(cands) > 0 {
			best := ""
			for _, c := range cands {
				if len(c) > len(best) {
					best = c
				}
			}
			name = strings.TrimSpace(best)
		}
		out.Members = append(out.Members, KKMember{Name: name, NIK: nik})
	}

	filled, total := 0.0, 5.0
	if out.NoKK != "" {
		filled++
	}
	if out.HeadName != "" {
		filled++
	}
	if out.Address != "" {
		filled++
	}
	if len(out.Members) > 0 {
		filled += 2
	}
	out.Confidence = filled / total
	if out.Confidence < 0.65 {
		out.NeedsReview = append(out.NeedsReview, "kualitas baca rendah — cocokkan dengan fisik KK")
	}
	return out
}

// ToMap meratakan saran KK ke fields JSON (prefix kk_).
func (k KKSuggestion) ToMap() map[string]string {
	parts := make([]string, 0, len(k.Members))
	for _, m := range k.Members {
		if m.Name != "" {
			parts = append(parts, m.Name+" ("+m.NIK+")")
		} else {
			parts = append(parts, m.NIK)
		}
	}
	m := map[string]string{
		"kk_no":        k.NoKK,
		"kk_no_valid":  strconv.FormatBool(k.NoKKValid),
		"kk_no_issue":  k.NoKKIssue,
		"kk_kepala":    k.HeadName,
		"kk_alamat":    k.Address,
		"kk_rt":        k.RT,
		"kk_rw":        k.RW,
		"kk_desa":      k.Village,
		"kk_kecamatan": k.District,
		"kk_anggota":   strings.Join(parts, "; "),
		"kk_jumlah":    strconv.Itoa(len(k.Members)),
		"kk_conf":      fmt.Sprintf("%.2f", k.Confidence),
		"kk_review":    strings.Join(k.NeedsReview, " | "),
	}
	for i, mem := range k.Members {
		n := strconv.Itoa(i + 1)
		m["kk_nama_"+n] = mem.Name
		m["kk_nik_"+n] = mem.NIK
	}
	return m
}

// --- NPWP ---

// NPWPSuggestion adalah kandidat prefill NPWP (review-only).
type NPWPSuggestion struct {
	NPWP        string // digit saja (15 atau 16)
	Formatted   string // tampilan bertitik untuk 15 digit
	NPWPValid   bool
	NPWPIssue   string
	Name        string // nama wajib pajak
	Address     string
	Confidence  float64
	NeedsReview []string
	RawText     string
}

var (
	reNPWPLabel = regexp.MustCompile(`(?i)NPWP\D{0,10}([\d][\d\.\s\-]{13,30}\d)`)
	reNPWPStd   = regexp.MustCompile(`\b\d{2}\.\d{3}\.\d{3}\.\d\-\d{3}\.\d{3}\b`)
	reNPWP15    = regexp.MustCompile(`\b\d{15}\b`)
	reWPName    = regexp.MustCompile(`(?i)NAMA(?:\s+WAJIB\s+PAJAK|\s+WP)?\s*[:\-]?\s*([A-Z][A-Z \.'\-]{2,60})`)
	reWPAddr    = regexp.MustCompile(`(?i)ALAMAT\s*[:\-]?\s*([A-Z0-9 \.\-/]{3,80})`)
)

// ValidateNPWP memeriksa 15 digit (format 2.3.3-1.3.3) atau 16 digit (format baru).
func ValidateNPWP(npwp string) (bool, string) {
	d := reDigitsOnly.ReplaceAllString(npwp, "")
	switch len(d) {
	case 15:
		if d == strings.Repeat("0", 15) {
			return false, "NPWP nol semua"
		}
		return true, ""
	case 16:
		if d == strings.Repeat("0", 16) {
			return false, "NPWP nol semua"
		}
		return true, ""
	default:
		return false, "harus 15 digit (format 99.999.999.9-999.999) atau 16 digit"
	}
}

// FormatNPWP15 menampilkan 15 digit sebagai 99.999.999.9-999.999.
func FormatNPWP15(digits string) string {
	d := reDigitsOnly.ReplaceAllString(digits, "")
	if len(d) != 15 {
		return digits
	}
	return fmt.Sprintf("%s.%s.%s.%s-%s.%s", d[0:2], d[2:5], d[5:8], d[8:9], d[9:12], d[12:15])
}

// ParseNPWP mengekstrak saran prefill dari teks OCR kartu/surat NPWP.
func ParseNPWP(text string) NPWPSuggestion {
	out := NPWPSuggestion{RawText: text}
	t := strings.ToUpper(text)

	if m := reNPWPLabel.FindStringSubmatch(t); len(m) > 1 {
		d := reDigitsOnly.ReplaceAllString(m[1], "")
		switch {
		case len(d) >= 16:
			out.NPWP = d[:16]
		case len(d) >= 15:
			out.NPWP = d[:15]
		}
	}
	if out.NPWP == "" {
		if m := reNPWPStd.FindString(t); m != "" {
			out.NPWP = reDigitsOnly.ReplaceAllString(m, "")
		} else if m := reNPWP15.FindString(t); m != "" {
			out.NPWP = m
		} else if m := reNIK.FindString(t); m != "" && strings.Contains(t, "NPWP") {
			out.NPWP = m // format baru 16 digit tanpa label jelas
			out.NeedsReview = append(out.NeedsReview, "NPWP tanpa label jelas — pastikan digit tidak tertukar NIK")
		}
	}
	if out.NPWP != "" {
		ok, issue := ValidateNPWP(out.NPWP)
		out.NPWPValid = ok
		out.NPWPIssue = issue
		if !ok {
			out.NeedsReview = append(out.NeedsReview, "NPWP tidak valid: "+issue)
		} else if len(out.NPWP) == 15 {
			out.Formatted = FormatNPWP15(out.NPWP)
		} else {
			out.Formatted = out.NPWP
		}
	} else {
		out.NeedsReview = append(out.NeedsReview, "NPWP tidak terbaca")
	}

	if m := reWPName.FindStringSubmatch(t); len(m) > 1 {
		out.Name = cleanLine(m[1])
	} else {
		lines := strings.Split(t, "\n")
		for i, ln := range lines {
			if strings.Contains(ln, "NAMA") && i+1 < len(lines) {
				if c := cleanLine(lines[i+1]); len(c) >= 3 {
					out.Name = c
					break
				}
			}
		}
	}
	if m := reWPAddr.FindStringSubmatch(t); len(m) > 1 {
		out.Address = cleanLine(m[1])
	}

	filled, total := 0.0, 3.0
	if out.NPWP != "" {
		filled++
	}
	if out.Name != "" {
		filled++
	}
	if out.Address != "" {
		filled++
	}
	out.Confidence = filled / total
	if out.Confidence < 0.65 {
		out.NeedsReview = append(out.NeedsReview, "kualitas baca rendah — cocokkan dengan fisik kartu NPWP")
	}
	return out
}

// ToMap meratakan saran NPWP ke fields JSON (prefix npwp_).
func (n NPWPSuggestion) ToMap() map[string]string {
	return map[string]string{
		"npwp_no":        n.NPWP,
		"npwp_formatted": n.Formatted,
		"npwp_valid":     strconv.FormatBool(n.NPWPValid),
		"npwp_issue":     n.NPWPIssue,
		"npwp_nama":      n.Name,
		"npwp_alamat":    n.Address,
		"npwp_conf":      fmt.Sprintf("%.2f", n.Confidence),
		"npwp_review":    strings.Join(n.NeedsReview, " | "),
	}
}
