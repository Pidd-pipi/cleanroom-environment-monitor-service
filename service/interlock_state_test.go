package service

import (
	"testing"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

func newInterlockEnv(t *testing.T) (*config.Config, *store.Store, *Services) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewMemoryStore()
	za := domain.NewCleanZone("zone_a", "Zone A", "PA-X", domain.Iso5, domain.ProcessLithography)
	zb := domain.NewCleanZone("zone_b", "Zone B", "PA-X", domain.Iso6, domain.ProcessEtching)
	if _, err := st.CleanZones().Create(za); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CleanZones().Create(zb); err != nil {
		t.Fatal(err)
	}
	ma := domain.NewMonitorZone("mon_a", "zone_a", "Point A", "PC-A")
	mb := domain.NewMonitorZone("mon_b", "zone_b", "Point B", "PC-B")
	if _, err := st.MonitorZones().Create(ma); err != nil {
		t.Fatal(err)
	}
	if _, err := st.MonitorZones().Create(mb); err != nil {
		t.Fatal(err)
	}
	return cfg, st, New(cfg, st)
}

// TestRestoreNormalZoneRejected verifies a zone that is not interlocked
// cannot be restore-confirmed.
func TestRestoreNormalZoneRejected(t *testing.T) {
	_, _, svc := newInterlockEnv(t)
	if _, err := svc.Interlock.Restore("zone_a", "eng", "all clear", ""); err == nil {
		t.Fatal("restoring a normal zone must be rejected")
	}
}

// TestRestoreClosesAllAreaLogs verifies restoring one zone of a physical
// area closes every open interlock log of that area.
func TestRestoreClosesAllAreaLogs(t *testing.T) {
	_, st, svc := newInterlockEnv(t)
	if _, _, err := svc.Interlock.IssueForArea("zone_b", "mon_b", "manual", "", 2.0); err != nil {
		t.Fatal(err)
	}
	open, err := st.Interlocks().OpenCount()
	if err != nil {
		t.Fatal(err)
	}
	if open != 1 {
		t.Fatalf("want 1 open interlock log, got %d", open)
	}
	if _, err := svc.Interlock.Restore("zone_a", "eng", "all clear", ""); err != nil {
		t.Fatal(err)
	}
	open, err = st.Interlocks().OpenCount()
	if err != nil {
		t.Fatal(err)
	}
	if open != 0 {
		t.Fatalf("restoring a zone must close every open log of the physical area, still open: %d", open)
	}
}
