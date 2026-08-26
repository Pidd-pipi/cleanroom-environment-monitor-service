package domain

import (
	"testing"
	"time"
)

// TestEvaluateValidityNilMonitorNoPanic verifies a nil monitor is handled
// safely instead of crashing the caller.
func TestEvaluateValidityNilMonitorNoPanic(t *testing.T) {
	res := EvaluateValidity(nil, time.Now())
	if res.Valid {
		t.Fatal("nil monitor must not be considered valid")
	}
	if res.Reason == "" {
		t.Fatal("nil monitor must carry an invalid reason")
	}
}

// TestIsCalibrationExpiredZeroDue verifies a monitor without a calibration
// due date is never treated as calibration-expired.
func TestIsCalibrationExpiredZeroDue(t *testing.T) {
	m := NewMonitorZone("m1", "z1", "Point 1", "PC-1")
	m.Equipment.CalibrationDue = time.Time{}
	if m.IsCalibrationExpired(time.Now().Add(365 * 24 * time.Hour)) {
		t.Fatal("a zero calibration due date must not count as expired")
	}
}

// TestSetMaintenanceClearsSinceOnEnd verifies ending maintenance clears the
// maintenance-since marker.
func TestSetMaintenanceClearsSinceOnEnd(t *testing.T) {
	m := NewMonitorZone("m1", "z1", "Point 1", "PC-1")
	m.SetMaintenance(true, "pm round")
	if m.Equipment.MaintenanceSince == nil {
		t.Fatal("maintenance start must set the since marker")
	}
	m.SetMaintenance(false, "pm done")
	if m.Equipment.MaintenanceSince != nil {
		t.Fatal("ending maintenance must clear the maintenance-since marker")
	}
}

// TestEffectiveEnvRangeFallsBackToProcessDefaults verifies a monitor without
// explicit overrides inherits the process default environment range.
func TestEffectiveEnvRangeFallsBackToProcessDefaults(t *testing.T) {
	m := NewMonitorZone("m1", "z1", "Point 1", "PC-1")
	r := m.EffectiveEnvRange(ProcessEtching)
	want := ProcessDefaultsFor(ProcessEtching)
	if r.TempMin != want.TempMin || r.TempMax != want.TempMax {
		t.Fatalf("expected process default temperature range %v..%v, got %v..%v", want.TempMin, want.TempMax, r.TempMin, r.TempMax)
	}
	if r.HumidityMin != want.HumidityMin || r.HumidityMax != want.HumidityMax {
		t.Fatalf("expected process default humidity range %v..%v, got %v..%v", want.HumidityMin, want.HumidityMax, r.HumidityMin, r.HumidityMax)
	}
	if r.PressureMin != want.PressureMin || r.PressureMax != want.PressureMax {
		t.Fatalf("expected process default pressure range %v..%v, got %v..%v", want.PressureMin, want.PressureMax, r.PressureMin, r.PressureMax)
	}
}
