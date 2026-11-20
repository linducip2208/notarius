package docengine

import "testing"

func TestRenderAndMissing(t *testing.T) {
	tpl := "Saya {{notary_name}}, berkantor di {{notary_office}}, menghadap {{client_name}}."
	vars := map[string]string{"notary_name": "Dr. A", "client_name": "Budi"}
	out := Render(tpl, vars)
	if out != "Saya Dr. A, berkantor di {{notary_office}}, menghadap Budi." {
		t.Fatalf("render=%q", out)
	}
	m := Missing(tpl, vars)
	if len(m) != 1 || m[0] != "notary_office" {
		t.Fatalf("missing=%v", m)
	}
	if h1, h2 := VersionHash(tpl, vars), VersionHash(tpl, vars); h1 != h2 {
		t.Fatal("hash not deterministic")
	}
	if w := Watermark("x"); w == "x" {
		t.Fatal("watermark must wrap")
	}
}
