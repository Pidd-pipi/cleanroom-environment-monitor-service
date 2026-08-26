package middleware

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"runtime/debug"
)

// PanicRecoveryMiddleware converts panics into unified 500 responses so a
// single bad request cannot take the whole service down.
func PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				slog.Error("middleware: panic recovered",
					"panic", rec,
					"request_id", RequestID(r.Context()),
					"path", r.URL.Path,
					"stack", string(debug.Stack()),
				)
				w.Header().Set("Content-Type", "application/json; charset=utf-8")
				w.WriteHeader(http.StatusInternalServerError)
				_ = json.NewEncoder(w).Encode(map[string]interface{}{
					"code":       500,
					"message":    "internal server error",
					"error":      "panic",
					"request_id": RequestID(r.Context()),
				})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
