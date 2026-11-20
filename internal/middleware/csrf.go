package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"net/http"

	"notaryos/internal/auth"
)

// CSRF protects state-changing form posts with a per-session token stored
// in a second cookie and compared against the hidden form field.
func CSRF(store auth.SessionStore, cookieName string) func(http.Handler) http.Handler {
	_ = store
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				next.ServeHTTP(w, r)
				return
			}
			if r.URL.Path == "/login" || r.URL.Path == "/logout" {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(cookieName + "_csrf")
			if err != nil {
				http.Error(w, "CSRF token hilang. Muat ulang halaman.", http.StatusForbidden)
				return
			}
			_ = r.ParseForm()
			if subtle.ConstantTimeCompare([]byte(r.FormValue("csrf")), []byte(cookie.Value)) != 1 {
				http.Error(w, "CSRF token tidak valid.", http.StatusForbidden)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// IssueCSRF creates/refreshes the CSRF cookie and returns the token for forms.
func IssueCSRF(w http.ResponseWriter, r *http.Request, cookieName string, secure bool) string {
	if c, err := r.Cookie(cookieName + "_csrf"); err == nil && c.Value != "" {
		return c.Value
	}
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	tok := base64.RawURLEncoding.EncodeToString(b)
	http.SetCookie(w, &http.Cookie{
		Name: cookieName + "_csrf", Value: tok, Path: "/",
		HttpOnly: false, SameSite: http.SameSiteStrictMode, Secure: secure, MaxAge: 43200,
	})
	return tok
}
