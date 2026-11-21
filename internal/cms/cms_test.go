package cms

import "testing"

func TestBrandingValidation(t *testing.T) {
	b := DefaultBranding()
	if b.Primary == "" || b.FontFamily == "" {
		t.Fatal("default harus lengkap")
	}
	if FontStack("x-exploit") != FontStack("system") {
		t.Fatal("font tak dikenal harus fallback")
	}
	if ButtonRadius("pill", 99) != "999px" {
		t.Fatal("pill harus 999px")
	}
	if ButtonRadius("rounded", 99) != "24px" {
		t.Fatal("radius harus dijepit 24")
	}
	if cleanColor("red\", \"x", "#000000") != "#000000" {
		t.Fatal("CSS injection harus ditolak")
	}
}

func TestNavURL(t *testing.T) {
	for _, bad := range []string{"", "javascript:alert(1)", "JaVaScRiPt:x", "data:text/html,x", "//evil.com", "ftp://x"} {
		if ValidNavURL(bad) {
			t.Fatalf("harus ditolak: %q", bad)
		}
	}
	for _, ok := range []string{"/tentang-kami", "https://kantor.id/a", "http://localhost:8080/x"} {
		if !ValidNavURL(ok) {
			t.Fatalf("harus lolos: %q", ok)
		}
	}
}

func TestSlug(t *testing.T) {
	if _, err := cleanSlug("Halo Dunia!"); err == nil {
		t.Fatal("slug kotor harus ditolak")
	}
	s, err := cleanSlug(" Layanan-PT ")
	if err != nil || s != "layanan-pt" {
		t.Fatalf("slug salah: %q %v", s, err)
	}
}

func TestSectionSchema(t *testing.T) {
	sec := &Section{Type: "nope", CTAURL: "javascript:x", Settings: map[string]any{"evil": 1, "columns": 9.0}}
	if err := ValidateSection(sec); err == nil {
		t.Fatal("tipe + CTA jahat harus ditolak")
	}
	sec = &Section{Type: "hero", Background: "x", Align: "x", Layout: "x", CTALabel: "Ayo", CTAURL: "/kontak",
		Settings: map[string]any{"columns": 2.0, "evil": 1, "ratio": "99:99"}}
	if err := ValidateSection(sec); err != nil {
		t.Fatal(err)
	}
	if sec.Background != "light" || sec.Align != "left" || len(sec.Settings) != 1 {
		t.Fatalf("normalisasi gagal: %+v", sec)
	}
}

func TestSanitize(t *testing.T) {
	out := SanitizeHTML(`<p>Halo</p><script>alert(1)</script><a href="javascript:x">klik</a><a href="/a">ok</a><img src="x" onerror="y"><b onclick="z">tebal</b>`)
	if out != `<p>Halo</p>alert(1)klik<a href="/a">ok</a><b>tebal</b>` {
		t.Fatalf("sanitasi salah: %q", out)
	}
	out = SanitizeHTML(`<img src="/media/1/a.png" alt="Logo">`)
	if out != `<img src="/media/1/a.png" alt="Logo" loading="lazy">` {
		t.Fatalf("img salah: %q", out)
	}
}

func TestMediaValidate(t *testing.T) {
	png := []byte("\x89PNG\r\n\x1a\n\x00\x00\x00\x0DIHDR")
	if _, err := ValidateMedia("x.png", 100, png); err != nil {
		t.Fatal(err)
	}
	if _, err := ValidateMedia("x.png", 100, []byte("GIF89a")); err == nil {
		t.Fatal("spoof png/gif harus ditolak")
	}
	if _, err := ValidateMedia("x.exe", 100, png); err == nil {
		t.Fatal("exe harus ditolak")
	}
	if _, err := ValidateMedia("x.svg", 100, []byte(`<svg onload="x">`)); err == nil {
		t.Fatal("svg aktif harus ditolak")
	}
}
