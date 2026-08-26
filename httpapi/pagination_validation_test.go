package httpapi

import (
	"net/http"
	"testing"
)

func TestAlertsPagination(t *testing.T) {
	ts, _, _ := newTestServer(t)
	for _, monitorID := range []string{"monitor_a1", "monitor_a2"} {
		_, env := doJSON(t, "POST", ts.URL+"/api/monitors/"+monitorID+"/samples", map[string]interface{}{
			"count_0_3um": 90000, "count_0_5um": 30000,
			"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
		})
		if env["code"] != float64(0) {
			t.Fatalf("ingest failed: %v", env)
		}
	}

	res, env := doJSON(t, "GET", ts.URL+"/api/alerts?limit=1&offset=0", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	data := env["data"].([]interface{})
	if len(data) != 1 {
		t.Fatalf("expected 1 item, got %d", len(data))
	}
	if got := res.Header.Get("X-Total-Count"); got != "2" {
		t.Fatalf("expected X-Total-Count=2, got %q", got)
	}
	if env["total"] == nil {
		t.Fatal("expected top-level total field")
	}
}

func TestListRejectsNegativeLimit(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, _ := doJSON(t, "GET", ts.URL+"/api/alerts?limit=-1", nil)
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative limit, got %d", res.StatusCode)
	}
}

func TestSampleRejectsNegativeCounts(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, _ := doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/samples", map[string]interface{}{
		"count_0_3um": -1, "count_0_5um": 0,
		"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for negative particle count, got %d", res.StatusCode)
	}
}
