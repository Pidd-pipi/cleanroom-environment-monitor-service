package domain

import (
	"testing"
	"time"
)

func TestNewSampleDefaults(t *testing.T) {
	s := NewSample("s1", "m1", 10, 20, 21.5, 45, 18, time.Time{})
	if !s.Valid {
		t.Fatal("new sample must be valid by default")
	}
	if s.Timestamp.IsZero() {
		t.Fatal("timestamp must be set")
	}
}

func TestMarkInvalid(t *testing.T) {
	s := NewSample("s1", "m1", 10, 20, 21, 45, 18, time.Now())
	s.MarkInvalid("calibration_expired")
	if s.Valid {
		t.Fatal("expected invalid")
	}
	if s.IsoClass != "" {
		t.Fatal("invalid sample must not carry an iso class")
	}
}

func TestSampleOutOfRange(t *testing.T) {
	s := NewSample("s1", "m1", 10, 20, 23.0, 70.0, 3.0, time.Now())
	r := EnvRange{TempMin: 20, TempMax: 22, HumidityMin: 40, HumidityMax: 60, PressureMin: 10, PressureMax: 25}
	temp, hum, press := s.IsOutOfRange(r)
	if !temp || !hum || !press {
		t.Fatalf("expected all out of range, got temp=%v hum=%v press=%v", temp, hum, press)
	}
	s2 := NewSample("s2", "m1", 10, 20, 21.0, 50.0, 18.0, time.Now())
	temp, hum, press = s2.IsOutOfRange(r)
	if temp || hum || press {
		t.Fatalf("expected all in range, got temp=%v hum=%v press=%v", temp, hum, press)
	}
}

func TestSampleIsBelowLimit(t *testing.T) {
	s := NewSample("s1", "m1", 90000, 30000, 21, 45, 18, time.Now())
	lim := IsoLimit{Count0303: 100000, Count0505: 35000}
	if !s.IsBelowLimit(lim) {
		t.Fatal("sample below limit must report below")
	}
	s2 := NewSample("s2", "m1", 150000, 30000, 21, 45, 18, time.Now())
	if s2.IsBelowLimit(lim) {
		t.Fatal("sample above 0.3um limit must report above")
	}
}
