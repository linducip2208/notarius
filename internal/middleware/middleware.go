package middleware

import (
	"context"
	"net/http"

	"notaryos/internal/auth"
	"notaryos/internal/rbac"
)

type ctxKey string

const (
	CtxSession ctxKey = "sess"
	CtxUserID  ctxKey = "uid"
	CtxRole    ctxKey = "role"
)

// SecurityHeaders sets baseline headers (CSP allows Tabler + HTMX CDNs).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "SAMEORIGIN")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		// HSTS hanya bermakna di HTTPS; aman dikirim selalu, browser abaikan di HTTP.
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains; preload")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; style-src 'self' https://cdn.jsdelivr.net 'unsafe-inline'; "+
				"script-src 'self' https://cdn.jsdelivr.net https://unpkg.com 'unsafe-inline'; "+
				"img-src 'self' data:; font-src 'self' https://cdn.jsdelivr.net data:")
		next.ServeHTTP(w, r)
	})
}

// LoadSession attaches session info to context when cookie is valid.
func LoadSession(store auth.SessionStore, cookieName string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			c, err := r.Cookie(cookieName)
			if err == nil {
				if sess, ok := store.Get(c.Value); ok {
					ctx := r.Context()
					ctx = context.WithValue(ctx, CtxSession, sess)
					ctx = context.WithValue(ctx, CtxUserID, sess.UserID)
					ctx = context.WithValue(ctx, CtxRole, sess.Role)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RoleOf returns role from context, empty if anonymous.
func RoleOf(r *http.Request) string {
	if v := r.Context().Value(CtxRole); v != nil {
		return v.(string)
	}
	return ""
}

// RequireAuth redirects anonymous users to /login.
func RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if RoleOf(r) == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequirePerm enforces server-side RBAC (menu hiding is NOT security).
func RequirePerm(perm string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !rbac.Has(RoleOf(r), perm) {
			http.Error(w, "Forbidden: membutuhkan izin "+perm, http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
