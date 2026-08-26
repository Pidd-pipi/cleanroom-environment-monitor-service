package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

// RequestLogMiddleware emits one structured access-log line per request:
// method, path, query, status, duration, remote address and trace id.
func RequestLogMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		slog.Info("http.request",
			"method", r.Method,
			"path", r.URL.Path,
			"query", r.URL.RawQuery,
			"status", status,
			"duration_ms", time.Since(start).Milliseconds(),
			"bytes", rec.bytes,
			"remote", r.RemoteAddr,
			"request_id", RequestID(r.Context()),
		)
	})
}
