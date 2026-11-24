package notify

import "testing"

func TestValidatePushSubscription(t *testing.T) {
	ok := ValidatePushSubscription("https://fcm.googleapis.com/fake-endpoint-123", "p256dh-key", "auth-key")
	if ok != nil {
		t.Fatalf("endpoint valid ditolak: %v", ok)
	}
	cases := []struct {
		name     string
		endpoint string
		p256dh   string
		auth     string
	}{
		{"kosong", "", "k", "a"},
		{"bukan-url", "bukan url", "k", "a"},
		{"http-publik", "http://example.com/push/1", "k", "a"},
		{"tanpa-kunci", "https://example.com/push/1", "", ""},
	}
	for _, c := range cases {
		if err := ValidatePushSubscription(c.endpoint, c.p256dh, c.auth); err == nil {
			t.Fatalf("%s: seharusnya ditolak", c.name)
		}
	}
	// http localhost diizinkan untuk pengembangan lokal.
	if err := ValidatePushSubscription("http://localhost:8080/push/1", "k", "a"); err != nil {
		t.Fatalf("localhost http seharusnya diizinkan: %v", err)
	}
}

func TestTruncateRunes(t *testing.T) {
	if got := truncateRunes("héllo世界", 5); got != "héllo" {
		t.Fatalf("potong rune salah: %q", got)
	}
	if got := truncateRunes("abc", 10); got != "abc" {
		t.Fatalf("string pendek tidak boleh berubah: %q", got)
	}
}
