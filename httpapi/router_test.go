package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/service"
	"example.com/cleanroom-environment-monitor-service/store"
)

// newTestServer builds a router over a memory store with a fake web FS.
func newTestServer(t *testing.T) (*httptest.Server, *store.Store, *service.Services) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewMemoryStore()
	svc := service.New(cfg, st)
	boot := service.NewBootstrap(cfg, st, svc.Ingest)
	if err := boot.SeedIfEmpty(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	web := fstest.MapFS{
		"web/index.html": &fstest.MapFile{Data: []byte("<html>cleanroom monitor</html>")},
		"web/app.js":     &fstest.MapFile{Data: []byte("console.log('app')")},
	}
	router := NewRouter(cfg, st, svc, web)
	ts := httptest.NewServer(router.Handler())
	t.Cleanup(ts.Close)
	return ts, st, svc
}

func doJSON(t *testing.T, method, url string, body interface{}) (*http.Response, map[string]interface{}) {
	t.Helper()
	var reader io.Reader
	if body != nil {
		data, _ := json.Marshal(body)
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequest(method, url, reader)
	if err != nil {
		t.Fatalf("request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("do: %v", err)
	}
	defer res.Body.Close()
	var envelope map[string]interface{}
	_ = json.NewDecoder(res.Body).Decode(&envelope)
	return res, envelope
}

func TestHealthz(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, env := doJSON(t, "GET", ts.URL+"/api/healthz", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	if env["code"] != float64(0) {
		t.Fatalf("expected code 0, got %v", env["code"])
	}
}

func TestOverviewEndpoint(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, env := doJSON(t, "GET", ts.URL+"/api/overview", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200, got %d", res.StatusCode)
	}
	data := env["data"].(map[string]interface{})
	if data["total_zones"] != float64(3) {
		t.Fatalf("expected 3 zones, got %v", data["total_zones"])
	}
}

func TestSampleIngestChainHTTP(t *testing.T) {
	ts, _, _ := newTestServer(t)
	// Normal.
	_, env := doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/samples", map[string]interface{}{
		"count_0_3um": 30000, "count_0_5um": 8000,
		"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
	})
	if env["code"] != float64(0) {
		t.Fatalf("normal ingest failed: %v", env)
	}
	// Elevated.
	_, env = doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/samples", map[string]interface{}{
		"count_0_3um": 90000, "count_0_5um": 30000,
		"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
	})
	data := env["data"].(map[string]interface{})
	zone := data["zone"].(map[string]interface{})
	if zone["status"] != "elevated" {
		t.Fatalf("expected elevated, got %v", zone["status"])
	}
	// Over-limit -> interlock.
	res, env := doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/samples", map[string]interface{}{
		"count_0_3um": 140000, "count_0_5um": 50000,
		"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("over-limit ingest: %d %v", res.StatusCode, env)
	}
	data = env["data"].(map[string]interface{})
	if data["interlock_issued"] != true {
		t.Fatalf("expected interlock issued, got %v", data["interlock_issued"])
	}
}

func TestAlertsAckHTTP(t *testing.T) {
	ts, _, _ := newTestServer(t)
	_, _ = doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/samples", map[string]interface{}{
		"count_0_3um": 90000, "count_0_5um": 30000,
		"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
	})
	_, env := doJSON(t, "GET", ts.URL+"/api/alerts?type=particle", nil)
	data := env["data"].([]interface{})
	if len(data) == 0 {
		t.Fatal("expected particle alerts")
	}
	alert := data[0].(map[string]interface{})
	alertID := alert["id"].(string)

	// Ack without disposition must be a 400.
	res, _ := doJSON(t, "POST", ts.URL+"/api/alerts/"+alertID+"/ack", map[string]interface{}{
		"operator": "eng", "disposition": "",
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for missing disposition, got %d", res.StatusCode)
	}
	// Ack with disposition.
	_, env = doJSON(t, "POST", ts.URL+"/api/alerts/"+alertID+"/ack", map[string]interface{}{
		"operator": "eng_li", "disposition": "replaced filter",
	})
	if env["code"] != float64(0) {
		t.Fatalf("ack failed: %v", env)
	}
	ackData := env["data"].(map[string]interface{})
	if ackData["status"] != "acknowledged" {
		t.Fatalf("expected acknowledged, got %v", ackData["status"])
	}
}

func TestRestoreRequiresNoteHTTP(t *testing.T) {
	ts, _, _ := newTestServer(t)
	_, _ = doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/samples", map[string]interface{}{
		"count_0_3um": 140000, "count_0_5um": 50000,
		"temperature": 21.0, "humidity": 45.0, "pressure_diff": 20.0,
	})
	res, _ := doJSON(t, "POST", ts.URL+"/api/zones/zone_a1/restore", map[string]interface{}{
		"operator": "eng", "note": "",
	})
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for empty note, got %d", res.StatusCode)
	}
	_, env := doJSON(t, "POST", ts.URL+"/api/zones/zone_a1/restore", map[string]interface{}{
		"operator": "eng", "note": "cleaned and verified",
	})
	if env["code"] != float64(0) {
		t.Fatalf("restore failed: %v", env)
	}
}

func TestSPAServe(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatalf("get /: %v", err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", res.StatusCode)
	}
	if !strings.Contains(string(body), "cleanroom monitor") {
		t.Fatalf("unexpected body %q", body)
	}
	// SPA fallback for client routes.
	res2, err := http.Get(ts.URL + "/zones/zone_a1")
	if err != nil {
		t.Fatalf("get /zones/zone_a1: %v", err)
	}
	defer res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 for /zones/zone_a1, got %d", res2.StatusCode)
	}
	// Unknown API path must return JSON 404, not the SPA shell.
	res3, err := http.Get(ts.URL + "/api/nope")
	if err != nil {
		t.Fatalf("get /api/nope: %v", err)
	}
	defer res3.Body.Close()
	if res3.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for unknown api, got %d", res3.StatusCode)
	}
}

func TestMaintenanceHTTP(t *testing.T) {
	ts, _, _ := newTestServer(t)
	_, env := doJSON(t, "POST", ts.URL+"/api/monitors/monitor_a1/maintenance", map[string]interface{}{
		"in_maintenance": true, "note": "PM",
	})
	if env["code"] != float64(0) {
		t.Fatalf("maintenance failed: %v", env)
	}
	data := env["data"].(map[string]interface{})
	eq := data["equipment"].(map[string]interface{})
	if eq["in_maintenance"] != true {
		t.Fatalf("expected in_maintenance true, got %v", eq["in_maintenance"])
	}
}

func TestInterlockManualHTTP(t *testing.T) {
	ts, _, _ := newTestServer(t)
	_, env := doJSON(t, "POST", ts.URL+"/api/zones/zone_a1/interlock", map[string]interface{}{
		"reason": "manual_test", "ratio": 1.6,
	})
	if env["code"] != float64(0) {
		t.Fatalf("manual interlock failed: %v", env)
	}
	_, env = doJSON(t, "GET", ts.URL+"/api/interlocks", nil)
	data := env["data"].([]interface{})
	if len(data) == 0 {
		t.Fatal("expected interlock logs")
	}
}

func TestAuditEndpoint(t *testing.T) {
	ts, _, _ := newTestServer(t)
	_, env := doJSON(t, "GET", ts.URL+"/api/audit?limit=10", nil)
	if env["code"] != float64(0) {
		t.Fatalf("audit endpoint failed: %v", env)
	}
}

func TestMalformedJSON(t *testing.T) {
	ts, _, _ := newTestServer(t)
	res, err := http.Post(ts.URL+"/api/monitors/monitor_a1/samples", "application/json",
		strings.NewReader("{not json"))
	if err != nil {
		t.Fatalf("post: %v", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusBadRequest {
		t.Fatalf("expected 400 for malformed json, got %d", res.StatusCode)
	}
}

var _ = time.Now
var _ = domain.AlertParticle
