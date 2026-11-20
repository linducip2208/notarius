package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// trustedNets holds CIDR/IP ranges whose X-Forwarded-For we honor.
// Default: trust none — XFF from untrusted sources is ignored (anti-spoof).
var (
	trustedMu   sync.RWMutex
	trustedNets []*net.IPNet
	trustedIPs  []net.IP
)

// SetTrustedProxies configures which proxy addresses may set X-Forwarded-For.
// Entries may be CIDR ("127.0.0.1/32", "::1/128") or plain IP ("127.0.0.1").
func SetTrustedProxies(entries []string) {
	trustedMu.Lock()
	defer trustedMu.Unlock()
	trustedNets = nil
	trustedIPs = nil
	for _, e := range entries {
		e = strings.TrimSpace(e)
		if e == "" {
			continue
		}
		if strings.Contains(e, "/") {
			if _, n, err := net.ParseCIDR(e); err == nil {
				trustedNets = append(trustedNets, n)
			}
			continue
		}
		if ip := net.ParseIP(e); ip != nil {
			trustedIPs = append(trustedIPs, ip)
		}
	}
}

func remoteIP(r *http.Request) net.IP {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}
	return net.ParseIP(strings.TrimSpace(host))
}

func fromTrustedProxy(r *http.Request) bool {
	rip := remoteIP(r)
	if rip == nil {
		return false
	}
	trustedMu.RLock()
	defer trustedMu.RUnlock()
	for _, ip := range trustedIPs {
		if ip.Equal(rip) {
			return true
		}
	}
	for _, n := range trustedNets {
		if n.Contains(rip) {
			return true
		}
	}
	return false
}

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

// clientIP extracts the host part (no port), honoring X-Forwarded-For
// ONLY when the direct peer is an explicitly trusted proxy.
func clientIP(r *http.Request) string {
	if fromTrustedProxy(r) {
		if fwd := r.Header.Get("X-Forwarded-For"); fwd != "" {
			// Leftmost = original client; take first valid IP.
			for _, part := range strings.Split(fwd, ",") {
				if ip := strings.TrimSpace(part); ip != "" {
					return ip
				}
			}
		}
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
