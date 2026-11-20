package docx

import (
	"archive/zip"
	"bytes"
	"testing"
)

func TestExportValidZip(t *testing.T) {
	out := Export("Akta Uji", "Baris satu\nBaris dua", true)
	if len(out) < 4 || string(out[:2]) != "PK" {
		t.Fatal("docx harus berupa zip (PK)")
	}
	z, err := zip.NewReader(bytes.NewReader(out), int64(len(out)))
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range z.File {
		if f.Name == "word/document.xml" {
			found = true
			rc, _ := f.Open()
			buf := new(bytes.Buffer)
			buf.ReadFrom(rc)
			rc.Close()
			if !bytes.Contains(buf.Bytes(), []byte("DRAFT")) {
				t.Fatal("draft watermark hilang")
			}
		}
	}
	if !found {
		t.Fatal("word/document.xml hilang")
	}
}
