package verify

import (
	"strings"
	"testing"
)

func TestRoundTrip(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	code := Code(key, "INV", 42)
	if !strings.HasPrefix(code, "INV-42-") {
		t.Fatalf("format salah: %s", code)
	}
	kind, id, err := Parse(key, strings.ToLower(code))
	if err != nil {
		t.Fatalf("kode sendiri harus valid: %v", err)
	}
	if kind != "INV" || id != 42 {
		t.Fatalf("parse salah: %s %d", kind, id)
	}
}

func TestTamperRejected(t *testing.T) {
	key := []byte("0123456789abcdef0123456789abcdef")
	code := Code(key, "INV", 42)
	bad := code[:len(code)-1] + "X"
	if _, _, err := Parse(key, bad); err == nil {
		t.Fatal("kode modifikasi harus DITOLAK")
	}
	other := make([]byte, 32)
	if _, _, err := Parse(other, code); err == nil {
		t.Fatal("kunci berbeda harus DITOLAK")
	}
	if _, _, err := Parse(key, "sampah"); err == nil {
		t.Fatal("format sampah harus DITOLAK")
	}
}
