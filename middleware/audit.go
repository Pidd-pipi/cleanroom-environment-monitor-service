package middleware

import (
	"net/http"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// AuditLogger is the cross-cutting request audit middleware. Every request
// produces an http.request audit entry carrying method, path, status and
// latency.
type AuditLogger struct {
	store *store.Store
	// skipPaths lists prefixes that are not audited (health checks, static
	// assets) to keep the audit trail focused on business operations.
	skipPaths []string
}

// NewAuditLogger builds the audit middleware.
func NewAuditLogger(st *store.Store, skipPaths ...string) *AuditLogger {
	return &AuditLogger{store: st, skipPaths: skipPaths}
}

// Wrap returns the middleware handler.
func (m *AuditLogger) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := StartTime(r.Context())
		if start.IsZero() {
			start = time.Now()
		}
		rec := &responseRecorder{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(rec, r)

		path := r.URL.Path
		for _, skip := range m.skipPaths {
			if len(path) >= len(skip) && path[:len(skip)] == skip {
				return
			}
		}
		status := rec.status
		if status == 0 {
			status = http.StatusOK
		}
		latency := time.Since(start).Milliseconds()
		entry := domain.NewAuditEntry(
			m.store.NewID("audit"),
			domain.AuditHTTPRequest,
			"anonymous",
			"http",
			path,
			fmtDetail(r.Method, path, status, latency),
			time.Now().UTC(),
		)
		entry.RequestID = RequestID(r.Context())
		_, _ = m.store.Audit().Create(entry)
	})
}

// fmtDetail builds the audit detail line.
func fmtDetail(method, path string, status int, latencyMS int64) string {
	return method + " " + path + " -> " + itoa(status) + " (" + itoa64(latencyMS) + "ms)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	buf := make([]byte, 0, 8)
	for n > 0 {
		buf = append([]byte{byte('0' + n%10)}, buf...)
		n /= 10
	}
	if neg {
		buf = append([]byte{'-'}, buf...)
	}
	return string(buf)
}

func itoa64(n int64) string {
	return itoa(int(n))
}
