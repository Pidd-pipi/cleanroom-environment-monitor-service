package service

import (
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// TestIngestLithoHumidityTriggersAlert verifies a lithography zone uses its
// own process thresholds: humidity 55% is out of the litho range and must
// raise a temp_humidity alert.
func TestIngestLithoHumidityTriggersAlert(t *testing.T) {
	cfg := newIngestCfg()
	st := newIngestStore(t, "z1", domain.Iso5, domain.ProcessLithography)
	svc := New(cfg, st)
	res, err := svc.Ingest.Process(IngestRequest{
		MonitorZoneID: "m1",
		Count0303:     30000,
		Count0505:     8000,
		Temperature:   21.0,
		Humidity:      55.0,
		PressureDiff:  20.0,
		Timestamp:     time.Now().UTC(),
		Operator:      "test",
	})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, a := range res.AlertsCreated {
		if a.Type == domain.AlertTempHumidity {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("humidity 55% on a litho zone (range 40..50) must create a temp_humidity alert")
	}
}
