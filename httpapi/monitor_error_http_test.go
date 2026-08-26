package httpapi

import (
	"net/http"
	"testing"
)

// TestGetMissingMonitorReturns404 verifies the HTTP contract for an
// unknown monitor zone: 404, never a generic 500.
func TestGetMissingMonitorReturns404(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, _ := doJSON(t, http.MethodGet, ts.URL+"/api/monitors/nope", nil)
	if res.StatusCode != http.StatusNotFound {
		t.Fatalf("GET missing monitor: want 404, got %d", res.StatusCode)
	}
}

// TestCreateDuplicateCounterReturns409 verifies that reusing an already
// assigned particle counter id yields a 409 conflict.
func TestCreateDuplicateCounterReturns409(t *testing.T) {
	ts, _, _ := newTestServer(t)
	body := map[string]interface{}{
		"id":                 "m_dup",
		"clean_zone_id":      "zone_a1",
		"name":               "Dup Point",
		"particle_counter_id": "PC-1001",
	}
	res, _ := doJSON(t, http.MethodPost, ts.URL+"/api/monitors", body)
	if res.StatusCode != http.StatusConflict {
		t.Fatalf("POST duplicate counter: want 409, got %d", res.StatusCode)
	}
}
