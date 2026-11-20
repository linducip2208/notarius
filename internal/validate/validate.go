package validate

import (
	"errors"
	"net/mail"
	"regexp"
	"strings"
)

var (
	nikRe   = regexp.MustCompile(`^\d{16}$`)
	phoneRe = regexp.MustCompile(`^\+?[\d][\d\s\-]{7,18}$`)
	npwpRe  = regexp.MustCompile(`^\d{15,16}$`)
)

// NIK validates Indonesian NIK: exactly 16 digits.
func NIK(s string) error {
	if !nikRe.MatchString(strings.TrimSpace(s)) {
		return errors.New("nik must be exactly 16 digits")
	}
	return nil
}

func Email(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("email is required")
	}
	_, err := mail.ParseAddress(s)
	if err != nil {
		return errors.New("invalid email address")
	}
	return nil
}

func Required(name, s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New(name + " is required")
	}
	return nil
}

func Phone(s string) error {
	if strings.TrimSpace(s) == "" {
		return errors.New("phone is required")
	}
	if !phoneRe.MatchString(strings.TrimSpace(s)) {
		return errors.New("invalid phone number")
	}
	return nil
}

// NPWP accepts 15 or 16 digit NPWP (new format 16 digits).
func NPWP(s string) error {
	if s == "" {
		return nil // optional
	}
	digits := regexp.MustCompile(`\D`).ReplaceAllString(s, "")
	if !npwpRe.MatchString(digits) {
		return errors.New("npwp must be 15 or 16 digits")
	}
	return nil
}
