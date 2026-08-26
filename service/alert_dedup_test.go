package service

import (
	"testing"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// TestAlertDedupMergeCountIncrements verifies that repeated occurrences
// within the dedup window keep accumulating on the same alert.
func TestAlertDedupMergeCountIncrements(t *testing.T) {
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
	svc := NewAlertService(cfg, st, NewAuditService(st))

	_, isNew1, err := svc.CreateWithDedup("z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "msg1", "s1", "")
	if err != nil || !isNew1 {
		t.Fatalf("first occurrence should create a new alert (isNew=%v err=%v)", isNew1, err)
	}
	_, isNew2, err := svc.CreateWithDedup("z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "msg2", "s2", "")
	if err != nil || isNew2 {
		t.Fatalf("second occurrence should merge into the existing alert (isNew=%v err=%v)", isNew2, err)
	}
	all, err := st.Alerts().List(store.AlertFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("want exactly 1 deduped alert, got %d", len(all))
	}
	if all[0].Count != 2 {
		t.Fatalf("dedup count must accumulate to 2, got %d", all[0].Count)
	}
}

// TestAlertDedupMergeLevelEscalates verifies a critical occurrence within
// the dedup window escalates the merged alert's level.
func TestAlertDedupMergeLevelEscalates(t *testing.T) {
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
	svc := NewAlertService(cfg, st, NewAuditService(st))
	_, _, err := svc.CreateWithDedup("z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "warn", "s1", "")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = svc.CreateWithDedup("z1", "m1", domain.AlertParticle, domain.AlertLevelCritical, "critical", "s2", "")
	if err != nil {
		t.Fatal(err)
	}
	all, err := st.Alerts().List(store.AlertFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 1 {
		t.Fatalf("want 1 deduped alert, got %d", len(all))
	}
	if all[0].Level != domain.AlertLevelCritical {
		t.Fatalf("merged alert must carry the escalated level, got %s", all[0].Level)
	}
}
