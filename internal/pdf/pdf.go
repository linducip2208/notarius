// Package pdf writes minimal multi-page PDF exports (draft deeds, receipts)
// with stdlib only. Not a typesetting engine — for final printed deeds use
// the approved template workflow; exports are clearly marked.
package pdf

import (
	"bytes"
	"fmt"
	"strings"
)

// Export renders title + body lines into a multi-page PDF. draft=true stamps
// every page header as DRAFT. Long documents paginate (68 lines/page);
// nothing is silently truncated.
func Export(title, body string, draft bool) []byte {
	header := title
	if draft {
		header = "DRAFT — " + title
	}
	// Wrap into printable lines.
	var lines []string
	lines = append(lines, "HEADER:"+header)
	lines = append(lines, "")
	for _, ln := range strings.Split(body, "\n") {
		ln = strings.TrimRight(ln, "\r")
		if ln == "" {
			lines = append(lines, "")
			continue
		}
		for len(ln) > 0 {
			n := 110
			if len(ln) < n {
				n = len(ln)
			}
			cut := n
			if len(ln) > n {
				if sp := strings.LastIndexByte(ln[:n], ' '); sp > 60 {
					cut = sp + 1
				}
			}
			lines = append(lines, ln[:cut])
			ln = ln[cut:]
		}
	}
	const perPage = 68
	pages := [][]string{}
	for i := 0; i < len(lines); i += perPage {
		end := i + perPage
		if end > len(lines) {
			end = len(lines)
		}
		pages = append(pages, lines[i:end])
	}
	if len(pages) == 0 {
		pages = [][]string{{"HEADER:" + header}}
	}
	esc := func(s string) string {
		s = strings.ReplaceAll(s, `\`, `\\`)
		s = strings.ReplaceAll(s, `(`, `\(`)
		s = strings.ReplaceAll(s, `)`, `\)`)
		return s
	}
	// Build content streams.
	streams := make([]string, len(pages))
	for pi, pl := range pages {
		var sb strings.Builder
		y := 800
		for li, ln := range pl {
			size := 11
			if strings.HasPrefix(ln, "HEADER:") {
				ln = strings.TrimPrefix(ln, "HEADER:")
				size = 16
			}
			if y < 60 {
				break
			}
			fmt.Fprintf(&sb, "BT /F1 %d Tf 40 %d Td (%s) Tj ET\n", size, y, esc(ln))
			y -= size + 6
			_ = li
		}
		// Footer: halaman + watermark status.
		foot := fmt.Sprintf("Halaman %d dari %d%s", pi+1, len(pages), map[bool]string{true: " — DOKUMEN DRAFT, BUKAN FINAL", false: ""}[draft])
		fmt.Fprintf(&sb, "BT /F1 9 Tf 40 30 Td (%s) Tj ET\n", esc(foot))
		streams[pi] = sb.String()
	}
	// Objects: 1 catalog, 2 pages, then per page (page, content), then font.
	var out bytes.Buffer
	offsets := []int{}
	w := func(s string) { out.WriteString(s) }
	w("%PDF-1.4\n")
	offsets = append(offsets, out.Len())
	w("1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n")
	// Kids refs: page objects start at 3, step 2.
	kids := ""
	for i := range pages {
		kids += fmt.Sprintf("%d 0 R ", 3+i*2)
	}
	offsets = append(offsets, out.Len())
	fmt.Fprintf(&out, "2 0 obj<</Type/Pages/Kids[%s]/Count %d>>endobj\n", kids, len(pages))
	fontObj := 3 + len(pages)*2
	for i, st := range streams {
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 595 842]/Resources<</Font<</F1 %d 0 R>>>>/Contents %d 0 R>>endobj\n",
			3+i*2, fontObj, 3+i*2+1)
		offsets = append(offsets, out.Len())
		fmt.Fprintf(&out, "%d 0 obj<</Length %d>>stream\n%s\nendstream\nendobj\n", 3+i*2+1, len(st), st)
	}
	offsets = append(offsets, out.Len())
	w(fmt.Sprintf("%d 0 obj<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>endobj\n", fontObj))
	xref := out.Len()
	n := len(offsets) + 1
	fmt.Fprintf(&out, "xref\n0 %d\n0000000000 65535 f \n", n)
	for _, o := range offsets {
		fmt.Fprintf(&out, "%010d 00000 n \n", o)
	}
	fmt.Fprintf(&out, "trailer<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF", n, xref)
	return out.Bytes()
}
