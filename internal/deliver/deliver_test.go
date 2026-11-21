package deliver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWebhookHandoff(t *testing.T) {
	var got map[string]any
	var sig string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sig = r.Header.Get("X-Signature")
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &got)
		w.WriteHeader(200)
	}))
	defer srv.Close()
	wh := Webhook{URL: srv.URL, Secret: "s3cr3t"}
	if err := wh.Send(7, Contact{Name: "Staf", Email: "s@x.id", Phone: "081"}, "T1", "B1"); err != nil {
		t.Fatal(err)
	}
	if got["title"] != "T1" || got["phone"] != "081" {
		t.Fatalf("payload salah: %v", got)
	}
	ts, _ := got["ts"].(string)
	mac := hmac.New(sha256.New, []byte("s3cr3t"))
	fmt.Fprintf(mac, "%d|%s|%s|%s", 7, "T1", "B1", ts)
	if sig != hex.EncodeToString(mac.Sum(nil)) {
		t.Fatal("HMAC tidak cocok — gateway harus menolak ini")
	}
	// server 500 → error jujur
	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()
	if err := (Webhook{URL: bad.URL}).Send(1, Contact{}, "T", "B"); err == nil {
		t.Fatal("status 500 harus error")
	}
}

func TestFromEnvEmpty(t *testing.T) {
	t.Setenv("NOTARYOS_NOTIFY_WEBHOOK", "")
	t.Setenv("NOTARYOS_SMTP_HOST", "")
	s := FromEnv()
	if err := s.Send(1, Contact{}, "T", "B"); err != nil {
		t.Fatal("tanpa env harus no-op sukses")
	}
	if !strings.Contains("nop", "nop") {
		t.Fatal("unreachable")
	}
}
