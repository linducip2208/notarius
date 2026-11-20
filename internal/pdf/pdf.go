// Package pdf writes minimal single-font PDF exports (draft deeds, receipts)
// with stdlib only. Not a typesetting engine — for final printed deeds use
// the approved template workflow; exports are clearly marked.
package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// Export renders title + body lines into a minimal PDF. draft=true stamps
// every page header as DRAFT.
func Export(title, body string, draft bool) []byte {
	var content bytes.Buffer
	y := 800
	header := title
	if draft {
		header = "DRAFT — " + title
	}
	writeLine := func(s string, size int) {
		if y < 60 {
			return // single page v1; overflow truncated honestly
		}
		esc := strings.ReplaceAll(s, `\`, `\\`)
		esc = strings.ReplaceAll(esc, `(`, `\(`)
		esc = strings.ReplaceAll(esc, `)`, `\)`)
		fmt.Fprintf(&content, "BT /F1 %d Tf 40 %d Td (%s) Tj ET\n", size, y, esc)
		y -= size + 6
	}
	writeLine(header, 16)
	y -= 8
	for _, ln := range strings.Split(body, "\n") {
		ln = strings.TrimRight(ln, "\r")
		for len(ln) > 110 {
			writeLine(ln[:110], 11)
			ln = ln[110:]
		}
		writeLine(ln, 11)
	}
	stream := content.String()
	var out bytes.Buffer
	offsets := []int{}
	w := func(s string) { out.WriteString(s) }
	w("%PDF-1.4\n")
	offsets = append(offsets, out.Len())
	w("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n")
	offsets = append(offsets, out.Len())
	w("2 0 obj<</Type/Pages/Kids[3 0 R]/Count 1>>endobj\n")
	offsets = append(offsets, out.Len())
	w("3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 595 842]/Resources<</Font<</F1 4 0 R>>>>/Contents 5 0 R>>endobj\n")
	offsets = append(offsets, out.Len())
	w("4 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n")
	offsets = append(offsets, out.Len())
	fmt.Fprintf(&out, "5 0 obj<</Length %d>>stream\n%s\nendstream\nendobj\n", len(stream), stream)
	xref := out.Len()
	n := len(offsets) + 1
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", n)
	for _, o := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF", n, xref)
	return out.Bytes()
}
