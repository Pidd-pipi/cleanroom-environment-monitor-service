package store

import (
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// TestAlertListFilterDoesNotCorruptStore verifies that a filtered read never
// rewrites the repository's internal alert array.
func TestAlertListFilterDoesNotCorruptStore(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	a1 := domain.NewAlert("a1", "z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "msg1", "s1", now.Add(-3*time.Minute))
	a2 := domain.NewAlert("a2", "z1", "m1", domain.AlertTempHumidity, domain.AlertLevelWarning, "msg2", "s1", now.Add(-2*time.Minute))
	a3 := domain.NewAlert("a3", "z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "msg3", "s1", now.Add(-time.Minute))
	for _, a := range []domain.CleanAlert{a1, a2, a3} {
		if _, err := st.Alerts().Create(a); err != nil {
			t.Fatal(err)
		}
	}

	filtered, err := st.Alerts().List(AlertFilter{Type: domain.AlertParticle})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("want 2 particle alerts, got %d", len(filtered))
	}

	all, err := st.Alerts().List(AlertFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("filtered read corrupted the store: want 3 alerts, got %d", len(all))
	}
	seen := map[string]bool{}
	for _, a := range all {
		seen[a.ID] = true
	}
	for _, id := range []string{"a1", "a2", "a3"} {
		if !seen[id] {
			t.Fatalf("alert %s disappeared after a filtered read", id)
		}
	}
}

// TestAlertListFilterNewestFirst verifies a filtered alert list is ordered
// newest first.
func TestAlertListFilterNewestFirst(t *testing.T) {
	st := NewMemoryStore()
	now := time.Now().UTC()
	a1 := domain.NewAlert("a1", "z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "msg1", "s1", now.Add(-3*time.Minute))
	a3 := domain.NewAlert("a3", "z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "msg3", "s1", now.Add(-time.Minute))
	for _, a := range []domain.CleanAlert{a1, a3} {
		if _, err := st.Alerts().Create(a); err != nil {
			t.Fatal(err)
		}
	}
	filtered, err := st.Alerts().List(AlertFilter{Type: domain.AlertParticle})
	if err != nil {
		t.Fatal(err)
	}
	if len(filtered) != 2 {
		t.Fatalf("want 2 alerts, got %d", len(filtered))
	}
	if filtered[0].ID != "a3" || filtered[1].ID != "a1" {
		t.Fatalf("filtered alerts must be newest first, got [%s %s]", filtered[0].ID, filtered[1].ID)
	}
}

// TestCleanZoneDeleteRemovesZone verifies a deleted clean zone never
// remains in the zone list.
func TestCleanZoneDeleteRemovesZone(t *testing.T) {
	st := NewMemoryStore()
	z1 := domain.NewCleanZone("z1", "Zone 1", "PA-A", domain.Iso5, domain.ProcessLithography)
	z2 := domain.NewCleanZone("z2", "Zone 2", "PA-A", domain.Iso6, domain.ProcessEtching)
	if _, err := st.CleanZones().Create(z1); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CleanZones().Create(z2); err != nil {
		t.Fatal(err)
	}
	if err := st.CleanZones().Delete("z1"); err != nil {
		t.Fatal(err)
	}
	all, err := st.CleanZones().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 || all[0].ID != "z2" {
		t.Fatalf("deleted zone still present after delete: %v", all)
	}
}

// TestCleanZoneDeleteCascadesMonitors verifies monitor zones of a deleted
// clean zone are removed with it.
func TestCleanZoneDeleteCascadesMonitors(t *testing.T) {
	st := NewMemoryStore()
	z := domain.NewCleanZone("z1", "Zone 1", "PA-A", domain.Iso5, domain.ProcessLithography)
	if _, err := st.CleanZones().Create(z); err != nil {
		t.Fatal(err)
	}
	m := domain.NewMonitorZone("m1", "z1", "Point 1", "PC-1")
	if _, err := st.MonitorZones().Create(m); err != nil {
		t.Fatal(err)
	}
	if err := st.CleanZones().Delete("z1"); err != nil {
		t.Fatal(err)
	}
	monitors, err := st.MonitorZones().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(monitors) != 0 {
		t.Fatalf("orphan monitor zones remain after clean zone delete: %v", monitors)
	}
}
