// Package cryptoxx provides AES-256-GCM field-level encryption for NIK data (UU PDP).
package cryptoxx

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"os"
)

// Key loads the 32-byte field-encryption key from NOTARYOS_FIELD_KEY (base64).
func Key() ([]byte, error) {
	raw := os.Getenv("NOTARYOS_FIELD_KEY")
	if raw == "" {
		return nil, errors.New("NOTARYOS_FIELD_KEY belum diset (base64, 32 byte)")
	}
	k, err := base64.StdEncoding.DecodeString(raw)
	if err != nil {
		return nil, err
	}
	if len(k) != 32 {
		return nil, errors.New("NOTARYOS_FIELD_KEY harus 32 byte")
	}
	return k, nil
}

// Encrypt seals plaintext with AES-256-GCM; output is base64(nonce|ciphertext).
func Encrypt(key, plaintext []byte) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, plaintext, nil)
	return base64.StdEncoding.EncodeToString(out), nil
}

// Decrypt opens output of Encrypt.
func Decrypt(key []byte, sealed string) ([]byte, error) {
	raw, err := base64.StdEncoding.DecodeString(sealed)
	if err != nil {
		return nil, err
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	if len(raw) < gcm.NonceSize() {
		return nil, errors.New("ciphertext terlalu pendek")
	}
	return gcm.Open(nil, raw[:gcm.NonceSize()], raw[gcm.NonceSize():], nil)
}
