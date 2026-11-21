package officex

import (
	"archive/zip"
	"bytes"
	"testing"
)

func zipBuf(t *testing.T, files map[string]string) *bytes.Reader {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range files {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	b := buf.Bytes()
	return bytes.NewReader(b)
}

func TestDocx(t *testing.T) {
	r := zipBuf(t, map[string]string{
		"word/document.xml": `<w:document xmlns:w="http://x"><w:body><w:p><w:r><w:t>Halo</w:t></w:r><w:r><w:t>Notaris</w:t></w:r></w:p></w:body></w:document>`,
	})
	text, ok := Extract("uji.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document", r, int64(r.Len()))
	if !ok {
		t.Fatal("docx harus terekstrak")
	}
	if text != "Halo Notaris" {
		t.Fatalf("teks salah: %q", text)
	}
}

func TestXlsx(t *testing.T) {
	r := zipBuf(t, map[string]string{
		"xl/sharedStrings.xml":     `<sst xmlns="http://x"><si><t>Nama</t></si><si><t>Budi</t></si></sst>`,
		"xl/worksheets/sheet1.xml": `<worksheet xmlns="http://x"><sheetData><row><c t="s"><v>0</v></c><c t="s"><v>1</v></c></row></sheetData></worksheet>`,
	})
	text, ok := Extract("uji.xlsx", "", r, int64(r.Len()))
	if !ok {
		t.Fatal("xlsx harus terekstrak")
	}
	if text != "Nama\tBudi" {
		t.Fatalf("teks salah: %q", text)
	}
}

func TestReject(t *testing.T) {
	r := zipBuf(t, map[string]string{"a.txt": "hi"})
	if _, ok := Extract("a.pdf", "application/pdf", r, int64(r.Len())); ok {
		t.Fatal("pdf harus ditolak")
	}
	if _, ok := Extract("rusak.docx", "", r, int64(r.Len())); ok {
		t.Fatal("zip rusak harus gagal jujur")
	}
}
