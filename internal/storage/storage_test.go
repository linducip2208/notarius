package storage

import (
	"io"
	"strings"
	"testing"
)

// zeroReader yields n zero bytes without allocating them.
type zeroReader struct{ n int64 }

func (z *zeroReader) Read(p []byte) (int, error) {
	if z.n <= 0 {
		return 0, io.EOF
	}
	k := int64(len(p))
	if k > z.n {
		k = z.n
	}
	z.n -= k
	return int(k), nil
}

func TestLocalOversizeRejected(t *testing.T) {
	l, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := l.Save("ok.bin", strings.NewReader("hello")); err != nil {
		t.Fatalf("small file must save: %v", err)
	}
	if err := l.Save("huge.bin", &zeroReader{n: (100 << 20) + 1}); err == nil {
		t.Fatal("file >100MB harus DITOLAK, bukan dipotong diam-diam")
	}
	if _, err := l.Open("huge.bin"); err == nil {
		t.Fatal("file raksasa tidak boleh tersisa di storage")
	}
}
