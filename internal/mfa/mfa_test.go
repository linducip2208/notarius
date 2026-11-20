package mfa

import (
	"strings"
	"testing"
	"time"
)

func TestTOTP(t *testing.T) {
	s, err := NewSecret()
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now()
	c, err := Code(s, now)
	if err != nil || len(c) != 6 {
		t.Fatalf("code=%q err=%v", c, err)
	}
	if !Verify(s, c, now) {
		t.Fatal("fresh code must verify")
	}
	if Verify(s, "000000", now) && c != "000000" {
		t.Fatal("wrong code must not verify")
	}
	_ = strings.ToLower(s)
}
