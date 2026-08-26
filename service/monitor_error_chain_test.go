package service

import (
	"errors"
	"testing"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

func newMonitorErrorEnv(t *testing.T) *ZoneService {
	t.Helper()
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
	return NewZoneService(st, NewAuditService(st))
}

func asDomain(t *testing.T, err error) *domain.Error {
	t.Helper()
	var de *domain.Error
	if !errors.As(err, &de) {
		t.Fatalf("error does not unwrap to a domain error: %v", err)
	}
	return de
}

// TestGetMonitorZoneMissingErrorsAsNotFound guards the error chain: a
// missing monitor zone must stay recognisable as not_found at the service
// boundary.
func TestGetMonitorZoneMissingErrorsAsNotFound(t *testing.T) {
	svc := newMonitorErrorEnv(t)
	_, err := svc.GetMonitorZone("nope")
	de := asDomain(t, err)
	if de.Code != domain.CodeNotFound {
		t.Fatalf("want not_found domain error, got code=%s", de.Code)
	}
}

// TestCreateMonitorZoneDuplicateErrorsAsConflict guards the error chain:
// a duplicate particle counter must stay recognisable as conflict.
func TestCreateMonitorZoneDuplicateErrorsAsConflict(t *testing.T) {
	svc := newMonitorErrorEnv(t)
	dup := domain.NewMonitorZone("m2", "z1", "Point 2", "PC-1")
	_, err := svc.CreateMonitorZone(dup, "eng", "")
	de := asDomain(t, err)
	if de.Code != domain.CodeConflict {
		t.Fatalf("want conflict domain error, got code=%s", de.Code)
	}
}
