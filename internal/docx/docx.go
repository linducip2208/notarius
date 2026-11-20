// Package docx writes minimal .docx exports with stdlib only (archive/zip).
// Honest minimal: title + paragraphs, DRAFT watermark as first paragraph.
package docx

import (
	"archive/zip"
	"bytes"
	"strings"
)

func esc(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

// Export renders title + body into a valid .docx (Word 2007+).
func Export(title, body string, draft bool) []byte {
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	add := func(name, content string) {
		f, _ := w.Create(name)
		f.Write([]byte(content))
	}
	add("[Content_Types].xml", `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/></Types>`)
	add("_rels/.rels", `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/></Relationships>`)
	var paras strings.Builder
	head := title
	if draft {
		head = "DRAFT — " + title
		paras.WriteString(`<w:p><w:pPr><w:jc w:val="center"/></w:pPr><w:r><w:rPr><w:b/><w:color w:val="FF0000"/></w:rPr><w:t>DRAFT — belum final, hanya dari template yang disetujui notaris</w:t></w:r></w:p>`)
	}
	paras.WriteString(`<w:p><w:r><w:rPr><w:b/><w:sz w:val="32"/></w:rPr><w:t>` + esc(head) + `</w:t></w:r></w:p>`)
	for _, ln := range strings.Split(body, "\n") {
		ln = strings.TrimRight(ln, "\r")
		if ln == "" {
			paras.WriteString(`<w:p/>`)
			continue
		}
		for len(ln) > 0 {
			chunk := ln
			if len(chunk) > 200 {
				chunk = chunk[:200]
			}
			ln = ln[len(chunk):]
			paras.WriteString(`<w:p><w:r><w:t xml:space="preserve">` + esc(chunk) + `</w:t></w:r></w:p>`)
		}
	}
	add("word/document.xml", `<?xml version="1.0" encoding="UTF-8"?><w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main"><w:body>`+paras.String()+`</w:body></w:document>`)
	w.Close()
	return buf.Bytes()
}
