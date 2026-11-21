// Package verify creates unforgeable public verification codes for invoices
// and receipts: INV-{id}-{checksum}, where checksum = HMAC-SHA256(fieldKey, id).
// No database column needed; verification recomputes the tag.
package verify

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Code returns a printable verification code for an invoice id.
func Code(key []byte, kind string, id int64) string {
	mac := hmac.New(sha256.New, key)
	fmt.Fprintf(mac, "%s:%d", kind, id)
	sum := hex.EncodeToString(mac.Sum(nil))
	return fmt.Sprintf("%s-%d-%s", strings.ToUpper(kind), id, strings.ToUpper(sum[:8]))
}

// Parse validates a code and returns kind + id.
func Parse(key []byte, code string) (string, int64, error) {
	parts := strings.Split(strings.TrimSpace(code), "-")
	if len(parts) != 3 {
		return "", 0, errors.New("format kode salah (contoh: INV-12-AB34CD56)")
	}
	kind := strings.ToUpper(parts[0])
	id, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || id <= 0 {
		return "", 0, errors.New("ID tidak valid")
	}
	want := Code(key, kind, id)
	if !hmac.Equal([]byte(strings.ToUpper(code)), []byte(want)) {
		return "", 0, errors.New("kode tidak cocok — bukan dokumen kami")
	}
	return kind, id, nil
}
