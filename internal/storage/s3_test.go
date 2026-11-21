package storage

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
)

// fakeS3 is an in-memory S3-ish endpoint: verifies SigV4 presence, stores bytes.
type fakeS3 struct {
	mu   sync.Mutex
	objs map[string][]byte
}

func (f *fakeS3) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	auth := r.Header.Get("Authorization")
	if !strings.HasPrefix(auth, "AWS4-HMAC-SHA256 ") || !strings.Contains(auth, "Signature=") {
		w.WriteHeader(403)
		return
	}
	if r.Header.Get("X-Amz-Date") == "" || r.Header.Get("X-Amz-Content-Sha256") == "" {
		w.WriteHeader(400)
		return
	}
	key := strings.TrimPrefix(r.URL.Path, "/bucket/")
	f.mu.Lock()
	defer f.mu.Unlock()
	switch r.Method {
	case "PUT":
		b, _ := io.ReadAll(r.Body)
		if f.objs == nil {
			f.objs = map[string][]byte{}
		}
		f.objs[key] = b
		w.WriteHeader(200)
	case "GET":
		b, ok := f.objs[key]
		if !ok {
			w.WriteHeader(404)
			return
		}
		w.WriteHeader(200)
		w.Write(b)
	default:
		w.WriteHeader(405)
	}
}

func TestS3RoundTrip(t *testing.T) {
	f := &fakeS3{}
	srv := httptest.NewServer(f)
	defer srv.Close()
	st, err := NewS3(S3Config{Endpoint: srv.URL, Region: "us-east-1", Bucket: "bucket", AccessKey: "AK", SecretKey: "SK"})
	if err != nil {
		t.Fatal(err)
	}
	if err := st.Save("o1/f1.bin", strings.NewReader("rahasia-notaris")); err != nil {
		t.Fatal(err)
	}
	rc, err := st.Open("o1/f1.bin")
	if err != nil {
		t.Fatal(err)
	}
	defer rc.Close()
	b, _ := io.ReadAll(rc)
	if string(b) != "rahasia-notaris" {
		t.Fatalf("roundtrip rusak: %q", b)
	}
	if _, err := st.Open("o1/tak-ada.bin"); err == nil {
		t.Fatal("objek hilang harus 404→error")
	}
}

func TestS3Incomplete(t *testing.T) {
	if _, err := NewS3(S3Config{}); err == nil {
		t.Fatal("konfig kosong harus ditolak")
	}
}
