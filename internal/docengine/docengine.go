// Package docengine renders deed/document drafts from approved templates.
// Variables use {{name}} placeholders. Final legal wording always comes
// from an approved template version; the engine never invents clauses.
package docengine

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

var varRe = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)

// ExtractVars lists distinct placeholder names in template order.
func ExtractVars(tpl string) []string {
	seen := map[string]bool{}
	var out []string
	for _, m := range varRe.FindAllStringSubmatch(tpl, -1) {
		if !seen[m[1]] {
			seen[m[1]] = true
			out = append(out, m[1])
		}
	}
	return out
}

// Missing returns placeholders with no value supplied.
func Missing(tpl string, vars map[string]string) []string {
	var out []string
	for _, v := range ExtractVars(tpl) {
		if _, ok := vars[v]; !ok {
			out = append(out, v)
		}
	}
	return out
}

// Render substitutes vars. Unknown placeholders are left intact so
// reviewers can see what is missing; Missing() reports them.
func Render(tpl string, vars map[string]string) string {
	return varRe.ReplaceAllStringFunc(tpl, func(m string) string {
		name := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(m, "{{"), "}}"))
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
}

// VersionHash identifies an exact template+vars rendering for the audit trail.
func VersionHash(tpl string, vars map[string]string) string {
	h := sha256.New()
	fmt.Fprintf(h, "tpl=%s\n", tpl)
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	for i := 0; i < len(keys); i++ {
		for j := i + 1; j < len(keys); j++ {
			if keys[j] < keys[i] {
				keys[i], keys[j] = keys[j], keys[i]
			}
		}
	}
	for _, k := range keys {
		fmt.Fprintf(h, "%s=%s\n", k, vars[k])
	}
	return fmt.Sprintf("%x", h.Sum(nil))[:16]
}

// Watermark wraps a draft rendering so it can never be mistaken for a final.
func Watermark(rendered string) string {
	return "================ DRAFT — BUKAN AKTA FINAL ================\n" + rendered +
		"\n================ DRAFT — BUKAN AKTA FINAL ================\n"
}
