package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"example.com/cleanroom-environment-monitor-service/store"
)

func TestRequestIDMiddleware(t *testing.T) {
	var got string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestID(r.Context())
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if got == "" {
		t.Fatal("request id must be injected")
	}
	if rec.Header().Get("X-Request-Id") == "" {
		t.Fatal("X-Request-Id header must be set")
	}
}

func TestRequestIDHonoursIncoming(t *testing.T) {
	var got string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = RequestID(r.Context())
	}))
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-Id", "trace-abc")
	handler.ServeHTTP(httptest.NewRecorder(), req)
	if got != "trace-abc" {
		t.Fatalf("expected trace-abc, got %q", got)
	}
}

func TestPanicRecoveryMiddleware(t *testing.T) {
	handler := PanicRecoveryMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	req := httptest.NewRequest("GET", "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "internal server error") {
		t.Fatalf("unexpected body %q", rec.Body.String())
	}
}

func TestAuditLoggerRecordsRequest(t *testing.T) {
	st := store.NewMemoryStore()
	logger := NewAuditLogger(st, "/api/healthz")
	handler := logger.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	req := httptest.NewRequest("POST", "/api/zones", nil)
	req = req.WithContext(WithRequestID(context.Background(), "req-test"))
	logger.Wrap(handler).ServeHTTP(httptest.NewRecorder(), req)

	entries, err := st.Audit().ListByAction("http.request")
	if err != nil {
		t.Fatalf("audit list: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("expected http.request audit entry")
	}
	if entries[0].RequestID != "req-test" {
		t.Fatalf("expected request id req-test, got %q", entries[0].RequestID)
	}
}

func TestAuditLoggerSkipsPaths(t *testing.T) {
	st := store.NewMemoryStore()
	logger := NewAuditLogger(st, "/api/healthz")
	handler := logger.Wrap(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	req := httptest.NewRequest("GET", "/api/healthz", nil)
	handler.ServeHTTP(httptest.NewRecorder(), req)
	entries, _ := st.Audit().ListByAction("http.request")
	if len(entries) != 0 {
		t.Fatalf("healthz must be skipped, got %d entries", len(entries))
	}
}
