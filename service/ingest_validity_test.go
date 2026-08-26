package service

import (
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// TestIngestValidSampleStaysValid verifies a healthy counter reading is not
// invalidated during ingestion.
func TestIngestValidSampleStaysValid(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewMemoryStore()
	z := domain.NewCleanZone("z1", "Zone 1", "PA-A", domain.Iso5, domain.ProcessLithography)
	if _, err := st.CleanZones().Create(z); err != nil {
		t.Fatal(err)
	}
	m := domain.NewMonitorZone("m1", "z1", "Point 1", "PC-1")
	if _, err := st.MonitorZones().Create(m); err != nil {
		t.Fatal(err)
	}
	svc := New(cfg, st)
	res, err := svc.Ingest.Process(IngestRequest{
		MonitorZoneID: "m1",
		Count0303:     30000,
		Count0505:     8000,
		Temperature:   21.0,
		Humidity:      45.0,
		PressureDiff:  20.0,
		Timestamp:     time.Now().UTC(),
		Operator:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !res.Sample.Valid {
		t.Fatalf("valid sample was marked invalid: %s", res.Sample.InvalidReason)
	}
}

func newIngestCfg() *config.Config {
	cfg := config.Default()
	cfg.DataFile = ""
	return cfg
}

func newIngestStore(t *testing.T, zoneID string, iso domain.IsoClass, proc domain.ProcessType) *store.Store {
	t.Helper()
	st := store.NewMemoryStore()
	z := domain.NewCleanZone(zoneID, "Zone", "PA-A", iso, proc)
	if _, err := st.CleanZones().Create(z); err != nil {
		t.Fatal(err)
	}
	m := domain.NewMonitorZone("m1", zoneID, "Point 1", "PC-1")
	if _, err := st.MonitorZones().Create(m); err != nil {
		t.Fatal(err)
	}
	return st
}
