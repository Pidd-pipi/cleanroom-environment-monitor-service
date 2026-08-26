package httpapi

import (
	"net/http"
	"testing"
)

// TestPostSampleAppendsOnce verifies a single sample POST persists exactly
// one sample for the monitor zone.
func TestPostSampleAppendsOnce(t *testing.T) {
	ts, st, _ := newTestServer(t)
	body := map[string]interface{}{
		"count_0_3um":   30000,
		"count_0_5um":   8000,
		"temperature":   21.0,
		"humidity":      45.0,
		"pressure_diff": 20.0,
		"timestamp":     "",
	}
	before, err := st.Samples().ListByMonitorZone("monitor_a1", 0)
	if err != nil {
		t.Fatal(err)
	}
	res, _ := doJSON(t, http.MethodPost, ts.URL+"/api/monitors/monitor_a1/samples", body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("POST sample: want 200, got %d", res.StatusCode)
	}
	samples, err := st.Samples().ListByMonitorZone("monitor_a1", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(samples) != len(before)+1 {
		t.Fatalf("one POST must persist exactly one extra sample, got %d -> %d", len(before), len(samples))
	}
}

// TestPaginateClampsHugeOffset verifies an oversized pagination offset
// returns an empty page instead of panicking.
func TestPaginateClampsHugeOffset(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, _ := doJSON(t, http.MethodGet, ts.URL+"/api/zones?limit=10&offset=999999", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("huge offset must not panic: want 200, got %d", res.StatusCode)
	}
}
