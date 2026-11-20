package cryptoxx

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"os"
	"testing"
)

func TestRoundtrip(t *testing.T) {
	k := make([]byte, 32)
	_, _ = rand.Read(k)
	os.Setenv("NOTARYOS_FIELD_KEY", base64.StdEncoding.EncodeToString(k))
	key, err := Key()
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := Encrypt(key, []byte("3174051209900001"))
	if err != nil {
		t.Fatal(err)
	}
	plain, err := Decrypt(key, sealed)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(plain, []byte("3174051209900001")) {
		t.Fatal("roundtrip broken")
	}
	if sealed == "3174051209900001" {
		t.Fatal("must not store plaintext")
	}
}
