package config

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Config holds all runtime configuration. Secrets come from env only.
type Config struct {
	Addr           string
	DSN            string
	SessionName    string
	TLSCookie      bool
	StorageDir     string
	AppBaseURL     string
	S3Endpoint     string
	S3Bucket       string
	AIBaseURL      string
	OCRBinary      string
	TrustedProxies []string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// getenvFirst returns the first non-empty env var (new canonical name wins,
// legacy alias still honoured so old deploys keep working).
func getenvFirst(def string, keys ...string) string {
	for _, k := range keys {
		if v := os.Getenv(k); v != "" {
			return v
		}
	}
	return def
}

func Load() Config {
	loadDotEnv(".env") // dev convenience; real env vars always win (never committed: .gitignore)
	port := getenv("PORT", "8080")
	host := getenv("DB_HOST", "127.0.0.1")
	dbport := getenv("DB_PORT", "3306")
	user := getenv("DB_USER", "notaryos")
	pass := os.Getenv("DB_PASS")
	name := getenv("DB_NAME", "notaryos")
	tls := getenv("DB_TLS", "false")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&tls=%s&interpolateParams=false",
		user, pass, host, dbport, name, tls)
	var proxies []string
	if v := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES")); v != "" {
		for _, p := range strings.Split(v, ",") {
			if s := strings.TrimSpace(p); s != "" {
				proxies = append(proxies, s)
			}
		}
	}
	return Config{
		Addr:           getenv("ADDR", ":"+port),
		DSN:            getenv("DB_DSN", dsn),
		SessionName:    getenv("SESSION_NAME", "notaryos_sess"),
		TLSCookie:      getenv("COOKIE_SECURE", "false") == "true",
		StorageDir:     getenv("STORAGE_DIR", "./storage"),
		AppBaseURL:     strings.TrimSuffix(getenv("APP_BASE_URL", ""), "/"),
		S3Endpoint:     getenvFirst("", "NOTARYOS_S3_ENDPOINT", "S3_ENDPOINT"),
		S3Bucket:       getenvFirst("", "NOTARYOS_S3_BUCKET", "S3_BUCKET"),
		AIBaseURL:      os.Getenv("AI_BASE_URL"),
		OCRBinary:      getenv("OCR_BINARY", ""),
		TrustedProxies: proxies,
	}
}

func MustInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}

// loadDotEnv fills missing env vars from a KEY=VALUE file (stdlib, no dependency).
// Existing environment always wins; missing file is silently ignored (prod uses real env).
func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.IndexByte(line, '='); i > 0 {
			k := strings.TrimSpace(line[:i])
			v := strings.TrimSpace(line[i+1:])
			if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
				v = v[1 : len(v)-1]
			}
			if k != "" && os.Getenv(k) == "" {
				_ = os.Setenv(k, v)
			}
		}
	}
}
