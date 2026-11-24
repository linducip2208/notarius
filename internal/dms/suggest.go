package dms

import "strings"

// SuggestCategory maps an OCR DocType (plus a PDF filename hint) to an
// existing office category. Matching is case-insensitive on the trimmed
// category name. It NEVER creates a category: when nothing matches it
// returns ok=false and callers render "—".
//
// Mapping (spec "Saran kategori otomatis saat upload"):
//
//	KTP   -> "Identitas"
//	KK    -> "Keluarga" (fallback "Identitas")
//	NPWP  -> "Pajak"
//	OTHER PDF whose filename mentions "akta" -> "Akta"
//	default -> no suggestion
//
// docType/mime are compared case-insensitively; the origName lookup for
// "akta" is a case-insensitive substring. Pure function (no DB writes) so
// it is safe to call on every dms_detail render and right after
// upload/OCR finishes.
func SuggestCategory(cats []Category, docType, mime, origName string) (hit Category, ok bool) {
	var wants []string
	switch strings.ToUpper(strings.TrimSpace(docType)) {
	case "KTP":
		wants = []string{"Identitas"}
	case "KK":
		// Prefer the more specific "Keluarga", fall back to "Identitas".
		wants = []string{"Keluarga", "Identitas"}
	case "NPWP":
		wants = []string{"Pajak"}
	default:
		// PDF akta heuristic: filename-based only, never content-based.
		if strings.EqualFold(strings.TrimSpace(mime), "application/pdf") &&
			strings.Contains(strings.ToLower(origName), "akta") {
			wants = []string{"Akta"}
		} else {
			return Category{}, false
		}
	}
	for _, w := range wants {
		for _, c := range cats {
			if strings.EqualFold(strings.TrimSpace(c.Name), w) {
				return c, true
			}
		}
	}
	return Category{}, false
}
