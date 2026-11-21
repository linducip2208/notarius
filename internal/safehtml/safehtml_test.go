package safehtml

import "testing"

func TestStripScript(t *testing.T) {
	out := Sanitize(`<p>Halo</p><script>alert(1)</script><b onclick="x()">tebal</b>`)
	if out != "<p>Halo</p>alert(1)<b>tebal</b>" {
		t.Fatalf("sanitasi salah: %q", out)
	}
}

func TestDropUnknown(t *testing.T) {
	out := Sanitize(`<div><a href="http://x">klik</a><img src=x onerror=y></div>`)
	if out != "klik" {
		t.Fatalf("tag asing harus dibuang isi dipertahankan: %q", out)
	}
}

func TestMalformed(t *testing.T) {
	out := Sanitize(`teks <buka dan <b>ok`)
	if out != "teks &lt;buka dan <b>ok" {
		t.Fatalf("rusak harus lolos aman: %q", out)
	}
}

func TestToText(t *testing.T) {
	if got := ToText(`<p>Halo <b>dunia</b></p><br><ul><li>satu</li></ul>`); got != "Halo dunia satu" {
		t.Fatalf("totext salah: %q", got)
	}
}
