// Package storage abstracts private file storage with an S3-compatible adapter.
package storage

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// FromEnv selects the backend: S3-compatible when NOTARYOS_S3_ENDPOINT is set,
// otherwise local directory. Keys embed office scope (unchanged semantics).
func FromEnv(storageDir string) (Store, error) {
	if ep := os.Getenv("NOTARYOS_S3_ENDPOINT"); ep != "" {
		return NewS3(S3Config{
			Endpoint:  ep,
			Region:    getenv("NOTARYOS_S3_REGION", "us-east-1"),
			Bucket:    os.Getenv("NOTARYOS_S3_BUCKET"),
			AccessKey: os.Getenv("NOTARYOS_S3_ACCESS"),
			SecretKey: os.Getenv("NOTARYOS_S3_SECRET"),
		})
	}
	return NewLocal(storageDir)
}

func getenv(k, d string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return d
}

// S3Config holds S3-compatible credentials (MinIO, AWS, dsb).
type S3Config struct {
	Endpoint, Region, Bucket, AccessKey, SecretKey string
}

// S3Store implements Store over S3 REST + SigV4 (PUT/GET/DELETE, path-style).
type S3Store struct {
	cfg S3Config
	cl  *http.Client
}

func NewS3(c S3Config) (*S3Store, error) {
	if c.Endpoint == "" || c.Bucket == "" || c.AccessKey == "" || c.SecretKey == "" {
		return nil, errors.New("konfigurasi S3 tidak lengkap (endpoint/bucket/access/secret)")
	}
	return &S3Store{cfg: c, cl: &http.Client{Timeout: 60 * time.Second}}, nil
}

func (s *S3Store) url(ref string) string {
	key := strings.TrimPrefix(filepath.ToSlash(filepath.Clean("/"+ref)), "/")
	return strings.TrimSuffix(s.cfg.Endpoint, "/") + "/" + s.cfg.Bucket + "/" + key
}

// sign4 builds SigV4 Authorization for one request.
func (s *S3Store) sign4(method, url, payloadHash string, t time.Time) (auth, amzDate string) {
	amzDate = t.UTC().Format("20060102T150405Z")
	dateStamp := t.UTC().Format("20060102")
	u := url[len(strings.TrimSuffix(s.cfg.Endpoint, "/"))+1:] // bucket/key...
	parts := strings.SplitN(u, "/", 2)
	uri := "/" + parts[0] + "/" + parts[1]
	host := hostOf(s.cfg.Endpoint)
	canonical := method + "\n" + uri + "\n\n" +
		"host:" + host + "\n" +
		"x-amz-content-sha256:" + payloadHash + "\n" +
		"x-amz-date:" + amzDate + "\n\n" +
		"host;x-amz-content-sha256;x-amz-date\n" + payloadHash
	scope := dateStamp + "/" + s.cfg.Region + "/s3/aws4_request"
	toSign := "AWS4-HMAC-SHA256\n" + amzDate + "\n" + scope + "\n" + sha256hex(canonical)
	kDate := hmacSHA256([]byte("AWS4"+s.cfg.SecretKey), dateStamp)
	kRegion := hmacSHA256(kDate, s.cfg.Region)
	kService := hmacSHA256(kRegion, "s3")
	kSigning := hmacSHA256(kService, "aws4_request")
	sig := hex.EncodeToString(hmacSHA256(kSigning, toSign))
	auth = "AWS4-HMAC-SHA256 Credential=" + s.cfg.AccessKey + "/" + scope +
		", SignedHeaders=host;x-amz-content-sha256;x-amz-date, Signature=" + sig
	return auth, amzDate
}

func hostOf(ep string) string {
	h := strings.TrimPrefix(strings.TrimPrefix(ep, "https://"), "http://")
	if i := strings.Index(h, "/"); i >= 0 {
		h = h[:i]
	}
	return h
}

func sha256hex(s string) string {
	sum := sha256.Sum256([]byte(s))
	return hex.EncodeToString(sum[:])
}

func hmacSHA256(key []byte, s string) []byte {
	m := hmac.New(sha256.New, key)
	m.Write([]byte(s))
	return m.Sum(nil)
}

func (s *S3Store) do(method, ref string, body io.Reader, bodyLen int64, payloadHash string) (*http.Response, error) {
	url := s.url(ref)
	var rdr io.Reader = body
	if rdr == nil {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, url, rdr)
	if err != nil {
		return nil, err
	}
	if bodyLen >= 0 {
		req.ContentLength = bodyLen
	}
	auth, amzDate := s.sign4(method, url, payloadHash, time.Now())
	req.Header.Set("Authorization", auth)
	req.Header.Set("X-Amz-Date", amzDate)
	req.Header.Set("X-Amz-Content-Sha256", payloadHash)
	return s.cl.Do(req)
}

// Save uploads ref (PUT). Body is buffered (≤100MB cap like local).
func (s *S3Store) Save(ref string, r io.Reader) error {
	buf, err := io.ReadAll(io.LimitReader(r, (100<<20)+1))
	if err != nil {
		return err
	}
	if int64(len(buf)) > 100<<20 {
		return errors.New("berkas melebihi batas 100MB")
	}
	sum := sha256.Sum256(buf)
	resp, err := s.do("PUT", ref, bytes.NewReader(buf), int64(len(buf)), hex.EncodeToString(sum[:]))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		return fmt.Errorf("s3 put status %d", resp.StatusCode)
	}
	return nil
}

// Open downloads ref (GET).
func (s *S3Store) Open(ref string) (io.ReadCloser, error) {
	resp, err := s.do("GET", ref, nil, 0, sha256hex(""))
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 {
		resp.Body.Close()
		return nil, fmt.Errorf("s3 get status %d", resp.StatusCode)
	}
	return resp.Body, nil
}
