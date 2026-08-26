package domain

import (
	"testing"
	"time"
)

func TestEvaluateValidityMaintenance(t *testing.T) {
	m := NewMonitorZone("m1", "z1", "point", "PC-1")
	m.SetMaintenance(true, "PM calibration")
	now := time.Now()
	res := EvaluateValidity(&m, now)
	if res.Valid {
		t.Fatal("sample during maintenance must be invalid")
	}
	if res.Reason != "counter_in_pm_maintenance" {
		t.Fatalf("unexpected reason %q", res.Reason)
	}
	m.SetMaintenance(false, "")
	res = EvaluateValidity(&m, now)
	if !res.Valid {
		t.Fatalf("sample after maintenance must be valid: %+v", res)
	}
}

func TestEvaluateValidityCalibrationExpired(t *testing.T) {
	m := NewMonitorZone("m1", "z1", "point", "PC-1")
	now := time.Now()
	m.Equipment.CalibrationDue = now.Add(-time.Hour)
	res := EvaluateValidity(&m, now)
	if res.Valid {
		t.Fatal("sample with expired calibration must be invalid")
	}
	if res.Reason != "calibration_expired" {
		t.Fatalf("unexpected reason %q", res.Reason)
	}
	m.Equipment.CalibrationDue = now.Add(time.Hour)
	res = EvaluateValidity(&m, now)
	if !res.Valid {
		t.Fatalf("sample before calibration expiry must be valid: %+v", res)
	}
}

func TestEvaluateValidityNilMonitor(t *testing.T) {
	res := EvaluateValidity(nil, time.Now())
	if res.Valid {
		t.Fatal("nil monitor must produce invalid sample")
	}
}

func TestEvaluateInvalidRatio(t *testing.T) {
	samples := []EnvSample{
		{Valid: true}, {Valid: false}, {Valid: false}, {Valid: true}, {Valid: false},
	}
	ratio, exceed := EvaluateInvalidRatio(samples, 5, 0.30)
	if ratio != 0.6 {
		t.Fatalf("expected ratio 0.6, got %v", ratio)
	}
	if !exceed {
		t.Fatal("ratio 0.6 must exceed 0.3 threshold")
	}
	ratio, exceed = EvaluateInvalidRatio(samples, 5, 0.7)
	if exceed {
		t.Fatal("ratio 0.6 must not exceed 0.7 threshold")
	}
	ratio, _ = EvaluateInvalidRatio(nil, 5, 0.3)
	if ratio != 0 {
		t.Fatal("empty window must produce 0 ratio")
	}
}

func TestValidSampleRatio(t *testing.T) {
	samples := []EnvSample{{Valid: true}, {Valid: false}}
	if got := ValidSampleRatio(samples); got != 0.5 {
		t.Fatalf("expected 0.5, got %v", got)
	}
	if got := ValidSampleRatio(nil); got != 0 {
		t.Fatalf("expected 0 for empty, got %v", got)
	}
}

func TestRecentSamplesBounds(t *testing.T) {
	samples := make([]EnvSample, 10)
	if got := RecentSamples(samples, 3); len(got) != 3 {
		t.Fatalf("expected 3, got %d", len(got))
	}
	if got := RecentSamples(samples, 0); len(got) != 10 {
		t.Fatalf("expected all 10, got %d", len(got))
	}
}
