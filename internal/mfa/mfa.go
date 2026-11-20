// Package mfa implements TOTP (RFC 6238, SHA-1, 30s, 6 digits) with stdlib only.
package mfa

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"time"
)

// NewSecret generates a random base32 secret for authenticator apps.
func NewSecret() (string, error) {
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// Code computes the TOTP value for secret at time t.
func Code(secret string, t time.Time) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(secret))
	if err != nil {
		return "", err
	}
	counter := uint64(t.Unix() / 30)
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], counter)
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	bin := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%06d", bin%1000000), nil
}

// Verify accepts current, previous, or next step (clock skew tolerance).
func Verify(secret, code string, now time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != 6 {
		return false
	}
	for _, d := range []int64{-30, 0, 30} {
		c, err := Code(secret, now.Add(time.Duration(d)*time.Second))
		if err != nil {
			return false
		}
		if hmac.Equal([]byte(c), []byte(code)) {
			return true
		}
	}
	return false
}

// ProvisioningURI builds the otpauth:// URI for QR codes.
func ProvisioningURI(secret, account, issuer string) string {
	return "otpauth://totp/" + issuer + ":" + account + "?secret=" + secret + "&issuer=" + issuer + "&digits=6&period=30"
}

var ErrInvalid = errors.New("kode MFA salah")
