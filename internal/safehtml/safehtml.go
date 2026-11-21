// Package safehtml sanitizes deed WYSIWYG HTML with a strict allowlist.
// Stored deed HTML is sanitized at write AND re-sanitized at render.
// PDF/DOCX/diff/search use ToText (tags stripped) so artifacts stay clean.
package safehtml

import (
	"html"
	"strings"
)

var allowed = map[string]bool{
	"p": true, "br": true, "b": true, "strong": true,
	"i": true, "em": true, "u": true,
	"ul": true, "ol": true, "li": true,
	"h3": true, "h4": true,
}

var voidTags = map[string]bool{"br": true}

// Sanitize strips everything except allowlisted tags (attributes always dropped).
// Malformed input degrades to escaped text — never raw passthrough.
func Sanitize(in string) string {
	var sb strings.Builder
	i := 0
	for i < len(in) {
		if in[i] != '<' {
			sb.WriteString(html.EscapeString(in[i : i+1]))
			i++
			continue
		}
		end := strings.IndexByte(in[i:], '>')
		if end < 0 {
			sb.WriteString(html.EscapeString(in[i:]))
			break
		}
		raw := strings.TrimSpace(in[i+1 : i+end])
		if strings.Contains(raw, "<") {
			// '<' di dalam tag = literal rusak; jangan makan tag berikutnya.
			sb.WriteString("&lt;")
			i++
			continue
		}
		i += end + 1
		if raw == "" {
			continue
		}
		closing := false
		if strings.HasPrefix(raw, "/") {
			closing = true
			raw = strings.TrimSpace(raw[1:])
		}
		// tag name = up to first space, slash, or end; lowercase it.
		name := raw
		for k, ch := range raw {
			if ch == ' ' || ch == '\t' || ch == '\n' || ch == '/' {
				name = raw[:k]
				break
			}
		}
		name = strings.ToLower(strings.TrimSuffix(name, "/"))
		if !allowed[name] {
			continue // drop tag AND its attributes entirely
		}
		if closing {
			sb.WriteString("</" + name + ">")
			continue
		}
		if voidTags[name] {
			sb.WriteString("<br>")
			continue
		}
		sb.WriteString("<" + name + ">")
	}
	return sb.String()
}

// ToText strips all tags to plain text (entities decoded, whitespace squeezed).
func ToText(in string) string {
	var sb strings.Builder
	inTag := false
	for i := 0; i < len(in); i++ {
		switch {
		case in[i] == '<':
			inTag = true
			sb.WriteByte(' ')
		case in[i] == '>':
			inTag = false
		case !inTag:
			sb.WriteByte(in[i])
		}
	}
	fields := strings.Fields(html.UnescapeString(sb.String()))
	return strings.Join(fields, " ")
}
