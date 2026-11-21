// Package officex extracts plain text from OOXML (docx/xlsx) for in-browser
// preview. Formatting is intentionally dropped — preview only, artifacts
// (PDF/DOCX) remain the source of truth.
package officex

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
)

const maxText = 50 << 10 // 50 KB cap

// Extract returns up to maxText plain-text characters from r based on name/MIME.
func Extract(name, mime string, r io.ReaderAt, size int64) (string, bool) {
	lower := strings.ToLower(name)
	mlower := strings.ToLower(mime)
	isDocx := strings.HasSuffix(lower, ".docx") || mlower == "application/vnd.openxmlformats-officedocument.wordprocessingml.document"
	isXlsx := strings.HasSuffix(lower, ".xlsx") || mlower == "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
	if !isDocx && !isXlsx {
		return "", false
	}
	zr, err := zip.NewReader(r, size)
	if err != nil {
		return "", false
	}
	open := func(path string) []byte {
		for _, f := range zr.File {
			if f.Name == path {
				rc, err := f.Open()
				if err != nil {
					return nil
				}
				defer rc.Close()
				b, _ := io.ReadAll(io.LimitReader(rc, maxText+1024))
				return b
			}
		}
		return nil
	}
	if isDocx {
		raw := open("word/document.xml")
		if raw == nil {
			return "", false
		}
		return truncate(collectText(raw, "t")), true
	}
	// xlsx: shared strings + first sheet inline values.
	shared := map[int]string{}
	if raw := open("xl/sharedStrings.xml"); raw != nil {
		type si struct {
			T []string `xml:"t"`
		}
		var ss struct {
			SI []si `xml:"si"`
		}
		if xml.Unmarshal(raw, &ss) == nil {
			for i, s := range ss.SI {
				shared[i] = strings.Join(s.T, "")
			}
		}
	}
	raw := open("xl/worksheets/sheet1.xml")
	if raw == nil {
		// fallback: dump shared strings only
		var sb strings.Builder
		for i := 0; i < len(shared); i++ {
			sb.WriteString(shared[i])
			sb.WriteString("\n")
		}
		if sb.Len() == 0 {
			return "", false
		}
		return truncate(sb.String()), true
	}
	type cell struct {
		T string `xml:"t,attr"`
		V string `xml:"v"`
	}
	var sh struct {
		Rows []struct {
			Cells []cell `xml:"c"`
		} `xml:"sheetData>row"`
	}
	var sb strings.Builder
	if xml.Unmarshal(raw, &sh) == nil {
		for _, row := range sh.Rows {
			var cols []string
			for _, c := range row.Cells {
				v := c.V
				if c.T == "s" {
					if idx, err := strconv.Atoi(strings.TrimSpace(v)); err == nil {
						v = shared[idx]
					}
				}
				cols = append(cols, v)
			}
			sb.WriteString(strings.Join(cols, "\t"))
			sb.WriteString("\n")
			if sb.Len() > maxText {
				break
			}
		}
	}
	if sb.Len() == 0 {
		return "", false
	}
	return truncate(sb.String()), true
}

// collectText concatenates <t>-ish leaf text (w:t) from document.xml.
func collectText(raw []byte, leaf string) string {
	dec := xml.NewDecoder(bytes.NewReader(raw))
	var sb strings.Builder
	for {
		tok, err := dec.Token()
		if err != nil {
			break
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == leaf {
			var inner struct {
				Text string `xml:",chardata"`
			}
			_ = dec.DecodeElement(&inner, &se)
			sb.WriteString(inner.Text)
			sb.WriteString(" ")
		}
		if sb.Len() > maxText {
			break
		}
	}
	return sb.String()
}

func truncate(s string) string {
	if len(s) > maxText {
		return s[:maxText] + "\n— dipotong 50 KB (pratinjau saja) —"
	}
	return strings.TrimSpace(s)
}
