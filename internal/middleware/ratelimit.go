package middleware

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// RateLimiter is a simple per-IP token bucket for sensitive endpoints (login).
type RateLimiter struct {
	mu     sync.Mutex
	hits   map[string][]time.Time
	max    int
	window time.Duration
}

func NewRateLimiter(max int, window time.Duration) *RateLimiter {
	return &RateLimiter{hits: map[string][]time.Time{}, max: max, window: window}
}

// clientIP extracts the host part (no port) preferring X-Forwarded-For.
func clientIP(r *http.Request) string {
	if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
		return fwd
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// Limit rejects with 429 when IP exceeds max requests per window.
func (l *RateLimiter) Limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := clientIP(r)
		now := time.Now()
		l.mu.Lock()
		recent := l.hits[ip][:0]
		for _, t := range l.hits[ip] {
			if now.Sub(t) < l.window {
				recent = append(recent, t)
			}
		}
		if len(recent) >= l.max {
			l.mu.Unlock()
			http.Error(w, "Terlalu banyak percobaan. Coba lagi nanti.", http.StatusTooManyRequests)
			return
		}
		l.hits[ip] = append(recent, now)
		l.mu.Unlock()
		next.ServeHTTP(w, r)
	})
}
