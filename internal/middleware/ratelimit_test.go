package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRateLimiter(t *testing.T) {
	lim := NewRateLimiter(3, time.Minute)
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(200) })
	h := lim.Limit(ok)
	codes := []int{}
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/login", nil)
		req.RemoteAddr = "10.0.0.1:1234"
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		codes = append(codes, rec.Code)
	}
	for i := 0; i < 3; i++ {
		if codes[i] != 200 {
			t.Fatalf("request %d should pass, got %d", i, codes[i])
		}
	}
	if codes[3] != 429 || codes[4] != 429 {
		t.Fatalf("expected 429s, got %v", codes)
	}
	// Different port, same IP → same bucket.
	req := httptest.NewRequest("POST", "/login", nil)
	req.RemoteAddr = "10.0.0.1:9999"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != 429 {
		t.Fatal("port must not create a new bucket")
	}
}
