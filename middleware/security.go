package middleware

import (
	"net/http"
	"strings"
)

// SecurityHeadersMiddleware sets baseline browser security headers on every
// response and disables caching for API payloads. A strict CSP is
// kept out of scope here so the embedded frontend keeps full control
// over its (non-inline) script/style setup.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		if strings.HasPrefix(r.URL.Path, "/api/") {
			h.Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}
