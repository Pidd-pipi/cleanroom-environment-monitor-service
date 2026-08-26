// Package middleware provides HTTP cross-cutting concerns: trace-id
// injection, operation audit logging and unified panic/error handling.
package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"time"
)

type ctxKey int

const (
	ctxKeyRequestID ctxKey = iota
	ctxKeyStartTime
)

const maxRequestIDLength = 128

// RequestID returns the trace id stored in the context ("" when absent).
func RequestID(ctx context.Context) string {
	if id, ok := ctx.Value(ctxKeyRequestID).(string); ok {
		return id
	}
	return ""
}

// StartTime returns the request start time stored by RequestIDMiddleware.
func StartTime(ctx context.Context) time.Time {
	if t, ok := ctx.Value(ctxKeyStartTime).(time.Time); ok {
		return t
	}
	return time.Time{}
}

// WithRequestID returns a context carrying the given trace id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, ctxKeyRequestID, id)
}

// RequestIDMiddleware injects a trace id into every request. An incoming
// X-Request-Id header is honoured when present and safe; otherwise a random
// trace id is generated. Each request gets its own derived context so trace
// ids and start times never leak across requests.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := sanitizeRequestID(r.Header.Get("X-Request-Id"))
		if id == "" {
			id = newRequestID()
		}
		ctx := context.WithValue(r.Context(), ctxKeyRequestID, id)
		ctx = context.WithValue(ctx, ctxKeyStartTime, time.Now())
		w.Header().Set("X-Request-Id", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// sanitizeRequestID rejects control characters and overlong values that
// could be used for header injection or log forging.
func sanitizeRequestID(v string) string {
	v = strings.TrimSpace(v)
	if v == "" || len(v) > maxRequestIDLength {
		return ""
	}
	for _, r := range v {
		if r < 0x20 || r == 0x7f {
			return ""
		}
	}
	return v
}

// newRequestID generates a short random hex trace id. It uses crypto/rand so
// concurrent requests are overwhelmingly unlikely to collide; the
// monotonic-nano fallback keeps ids unique even if the entropy source fails.
func newRequestID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("req-%x", time.Now().UnixNano())
	}
	return "req-" + hex.EncodeToString(b)
}
