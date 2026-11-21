package diff

import (
	"strings"
	"testing"
)

func TestCompareBasic(t *testing.T) {
	ops := Compare("a\nb\nc", "a\nB\nc\nd")
	var got []string
	for _, o := range ops {
		got = append(got, o.Kind+o.Line)
	}
	joined := strings.Join(got, "|")
	// Urutan -b/+B boleh tertukar (keduanya diff minimal yang valid).
	if joined != " a|-b|+B| c|+d" && joined != " a|+B|-b| c|+d" {
		t.Fatalf("diff salah: %v", got)
	}
	add, del := Stats(ops)
	if add != 2 || del != 1 {
		t.Fatalf("stat salah: +%d -%d", add, del)
	}
}

func TestCompareIdentical(t *testing.T) {
	ops := Compare("sama\npersis", "sama\npersis")
	for _, o := range ops {
		if o.Kind != " " {
			t.Fatalf("harus semua sama: %+v", ops)
		}
	}
}

func TestCompareEmpty(t *testing.T) {
	if ops := Compare("", "baru"); len(ops) != 1 || ops[0].Kind != "+" {
		t.Fatalf("tambah semua: %+v", ops)
	}
	if ops := Compare("lama", ""); len(ops) != 1 || ops[0].Kind != "-" {
		t.Fatalf("hapus semua: %+v", ops)
	}
}
