package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRequestIDNotReusedAcrossRequests verifies every request carries its
// own context values and never inherits a previous request's trace id.
func TestRequestIDNotReusedAcrossRequests(t *testing.T) {
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = w
		// capture what the downstream handler observes
	}))

	var observed []string
	handler = RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		observed = append(observed, RequestID(r.Context()))
	}))

	// Request 1
	req1 := httptest.NewRequest(http.MethodGet, "/a", nil)
	req1.Header.Set("X-Request-Id", "trace-first")
	handler.ServeHTTP(httptest.NewRecorder(), req1)

	// Request 2 must see its own trace id, not the first request's.
	req2 := httptest.NewRequest(http.MethodGet, "/b", nil)
	req2.Header.Set("X-Request-Id", "trace-second")
	handler.ServeHTTP(httptest.NewRecorder(), req2)

	if len(observed) != 2 {
		t.Fatalf("want 2 observations, got %d", len(observed))
	}
	if observed[0] != "trace-first" {
		t.Fatalf("first request saw %q", observed[0])
	}
	if observed[1] != "trace-second" {
		t.Fatalf("second request must carry its own trace id, saw %q", observed[1])
	}
}

// TestRequestIDUniquePerRequest verifies requests without an incoming id
// receive distinct generated trace ids.
func TestRequestIDUniquePerRequest(t *testing.T) {
	var ids []string
	handler := RequestIDMiddleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ids = append(ids, RequestID(r.Context()))
	}))
	for i := 0; i < 5; i++ {
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/x", nil))
	}
	if len(ids) != 5 {
		t.Fatalf("want 5 ids, got %d", len(ids))
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if id == "" {
			t.Fatal("empty trace id observed")
		}
		if seen[id] {
			t.Fatalf("trace id reused across requests: %q", id)
		}
		seen[id] = true
	}
}
