package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all runtime configuration. Secrets come from env only.
type Config struct {
	Addr        string
	DSN         string
	SessionName string
	TLSCookie   bool
	StorageDir  string
	S3Endpoint  string
	S3Bucket    string
	AIBaseURL   string
	OCRBinary   string
}

func getenv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func Load() Config {
	port := getenv("PORT", "8080")
	host := getenv("DB_HOST", "127.0.0.1")
	dbport := getenv("DB_PORT", "3306")
	user := getenv("DB_USER", "notaryos")
	pass := os.Getenv("DB_PASS")
	name := getenv("DB_NAME", "notaryos")
	tls := getenv("DB_TLS", "false")
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&loc=UTC&tls=%s&interpolateParams=false",
		user, pass, host, dbport, name, tls)
	return Config{
		Addr:        getenv("ADDR", ":"+port),
		DSN:         getenv("DB_DSN", dsn),
		SessionName: getenv("SESSION_NAME", "notaryos_sess"),
		TLSCookie:   getenv("COOKIE_SECURE", "false") == "true",
		StorageDir:  getenv("STORAGE_DIR", "./storage"),
		S3Endpoint:  os.Getenv("S3_ENDPOINT"),
		S3Bucket:    os.Getenv("S3_BUCKET"),
		AIBaseURL:   os.Getenv("AI_BASE_URL"),
		OCRBinary:   getenv("OCR_BINARY", ""),
	}
}

func MustInt(s string, def int) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
