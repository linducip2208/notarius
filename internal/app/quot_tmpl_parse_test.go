package app

import (
	"html/template"
	"testing"
	"time"
)

// TestQuotTemplatesParse memastikan gohtml (termasuk kartu Penawaran +
// halaman quotations) lolos parse seperti saat boot (embed ParseFS).
func TestQuotTemplatesParse(t *testing.T) {
	_, err := template.New("").Funcs(template.FuncMap{
		"can":       func(role, perm string) bool { return true },
		"wib":       func(t time.Time) string { return "" },
		"wibdate":   func(t time.Time) string { return "" },
		"css":       func(s string) template.CSS { return template.CSS(s) },
		"lower":     func(s string) string { return s },
		"hasPrefix": func(s, p string) bool { return false },
		"hasSuffix": func(s, p string) bool { return false },
		"maskNIK":   func(s string) string { return s },
		"redactNIK": func(s string) string { return s },
		"safeHTML":  func(s string) template.HTML { return template.HTML(s) },
		"cmsHTML":   func(s string) template.HTML { return template.HTML(s) },
		"plaintext": func(s string) string { return s },
		"sub":       func(a, b int64) int64 { return a - b },
		"refLink":   func(typ string, id int64) template.HTML { return "" },
	}).ParseFS(tmplFS, "templates/*.gohtml")
	if err != nil {
		t.Fatalf("template parse: %v", err)
	}
}
