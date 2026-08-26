package service

import (
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// newTestEnv builds a fully wired service container backed by a memory
// store and a fast-ticking config, ready for chain tests.
func newTestEnv(t *testing.T) (*Services, *store.Store, *config.Config) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = "" // keep tests off disk
	cfg.AutoInterlockAfter = 10 * time.Minute
	st := store.NewMemoryStore()
	svc := New(cfg, st)
	boot := NewBootstrap(cfg, st, svc.Ingest)
	if err := boot.SeedIfEmpty(); err != nil {
		t.Fatalf("bootstrap: %v", err)
	}
	return svc, st, cfg
}

// ingest posts a sample and fails the test on error.
func ingest(t *testing.T, svc *Services, monitorID string, c0303, c0505, temp, hum, press float64) IngestResult {
	t.Helper()
	res, err := svc.Ingest.Process(IngestRequest{
		MonitorZoneID: monitorID,
		Count0303:     c0303,
		Count0505:     c0505,
		Temperature:   temp,
		Humidity:      hum,
		PressureDiff:  press,
		Timestamp:     time.Now().UTC(),
		Operator:      "tester",
	})
	if err != nil {
		t.Fatalf("ingest %s: %v", monitorID, err)
	}
	return res
}

// iso5 limits for monitor_a1: 0.8 multiplier -> 80000 / 28000.
const (
	mA1Limit0303 = 80000.0
	mA1Limit0505 = 28000.0
)

func TestIngestChainNormalToInterlockAndRestore(t *testing.T) {
	svc, st, _ := newTestEnv(t)
	_ = st

	// 1) Normal sample.
	r1 := ingest(t, svc, "monitor_a1", 30000, 8000, 21.0, 45.0, 20.0)
	if !r1.Sample.Valid {
		t.Fatal("baseline sample must be valid")
	}
	if r1.Zone.Status != domain.ZoneStatusNormal {
		t.Fatalf("expected normal, got %s", r1.Zone.Status)
	}

	// 2) Elevated: ratio between 1.0 and 1.5.
	r2 := ingest(t, svc, "monitor_a1", 90000, 30000, 21.0, 45.0, 20.0) // ~1.125x
	if r2.Zone.Status != domain.ZoneStatusElevated {
		t.Fatalf("expected elevated, got %s", r2.Zone.Status)
	}
	if len(r2.AlertsCreated) == 0 || r2.AlertsCreated[0].Type != domain.AlertParticle {
		t.Fatalf("expected particle alert, got %+v", r2.AlertsCreated)
	}

	// 3) Over-limit: ratio >= 1.5 -> immediate area-wide interlock.
	r3 := ingest(t, svc, "monitor_a1", 140000, 50000, 21.0, 45.0, 20.0) // 1.75x
	if !r3.InterlockIssued {
		t.Fatal("expected interlock to be issued")
	}
	if r3.InterlockLog == nil {
		t.Fatal("expected interlock log")
	}
	zA, _ := svc.Zones.GetCleanZone("zone_a1")
	if zA.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("zone_a1 expected interlocked, got %s", zA.Status)
	}
	// Whole-area consistency: zone_b1 (same physical area PA-A) must also
	// be interlocked.
	zB, _ := svc.Zones.GetCleanZone("zone_b1")
	if zB.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("zone_b1 expected interlocked (area consistency), got %s", zB.Status)
	}
	// zone_c1 is in a different physical area and must NOT be interlocked.
	zC, _ := svc.Zones.GetCleanZone("zone_c1")
	if zC.Status == domain.ZoneStatusInterlocked {
		t.Fatal("zone_c1 must not be interlocked (different physical area)")
	}

	// 4) Recovery below limit keeps interlocked until manual restore.
	r4 := ingest(t, svc, "monitor_a1", 30000, 8000, 21.0, 45.0, 20.0)
	if r4.Zone.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("zone must stay interlocked until restore, got %s", r4.Zone.Status)
	}

	// 5) Manual restore confirmation moves the whole area to restored.
	restored, err := svc.Interlock.Restore("zone_a1", "eng_li", "filter cleaned", "")
	if err != nil {
		t.Fatalf("restore: %v", err)
	}
	if len(restored) != 2 {
		t.Fatalf("expected 2 restored zones, got %d", len(restored))
	}
	zA, _ = svc.Zones.GetCleanZone("zone_a1")
	if zA.Status != domain.ZoneStatusRestored {
		t.Fatalf("expected restored, got %s", zA.Status)
	}
	// 6) Next good sample returns to normal.
	r6 := ingest(t, svc, "monitor_a1", 30000, 8000, 21.0, 45.0, 20.0)
	if r6.Zone.Status != domain.ZoneStatusNormal {
		t.Fatalf("expected normal after restore + good sample, got %s", r6.Zone.Status)
	}
}

func TestIngestInvalidDuringMaintenanceAndDataQuality(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	// Put monitor_a1 into PM maintenance.
	if _, err := svc.Zones.SetMaintenance("monitor_a1", true, "PM check", "eng", ""); err != nil {
		t.Fatalf("set maintenance: %v", err)
	}
	r := ingest(t, svc, "monitor_a1", 30000, 8000, 21.0, 45.0, 20.0)
	if r.Sample.Valid {
		t.Fatal("sample during maintenance must be invalid")
	}
	if r.Sample.InvalidReason != "counter_in_pm_maintenance" {
		t.Fatalf("unexpected invalid reason %q", r.Sample.InvalidReason)
	}
	if !r.DataCredibilityLow {
		t.Fatal("100% invalid samples must flag low credibility")
	}
	found := false
	for _, a := range r.AlertsCreated {
		if a.Type == domain.AlertDataQuality {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected data_quality alert, got %+v", r.AlertsCreated)
	}
	// End maintenance and ingest a valid sample: data quality must resolve.
	_, _ = svc.Zones.SetMaintenance("monitor_a1", false, "", "eng", "")
	r2 := ingest(t, svc, "monitor_a1", 30000, 8000, 21.0, 45.0, 20.0)
	if !r2.Sample.Valid {
		t.Fatal("sample after maintenance must be valid")
	}
}

func TestAlertDedupWindow(t *testing.T) {
	svc, _, cfg := newTestEnv(t)
	// Two elevated samples within 20 minutes must merge.
	r1 := ingest(t, svc, "monitor_a1", 90000, 30000, 21.0, 45.0, 20.0)
	if len(r1.AlertsCreated) != 1 {
		t.Fatalf("expected 1 created alert, got %d", len(r1.AlertsCreated))
	}
	r2 := ingest(t, svc, "monitor_a1", 95000, 32000, 21.0, 45.0, 20.0)
	if len(r2.AlertsCreated) != 0 || len(r2.AlertsMerged) != 1 {
		t.Fatalf("second alert within dedup window must merge: created=%d merged=%d",
			len(r2.AlertsCreated), len(r2.AlertsMerged))
	}
	_ = cfg
}

func TestAlertAckAndEscalate(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	r := ingest(t, svc, "monitor_a1", 90000, 30000, 21.0, 45.0, 20.0)
	if len(r.AlertsCreated) == 0 {
		t.Fatal("expected alert")
	}
	alertID := r.AlertsCreated[0].ID
	// Ack requires disposition.
	if _, err := svc.Alerts.Ack(alertID, "eng", "", ""); err == nil {
		t.Fatal("ack without disposition must fail")
	}
	acked, err := svc.Alerts.Ack(alertID, "eng_wang", "wiped sensor lens", "")
	if err != nil {
		t.Fatalf("ack: %v", err)
	}
	if acked.Status != domain.AlertStatusAcknowledged {
		t.Fatalf("expected acknowledged, got %s", acked.Status)
	}
	// Escalating an acknowledged alert is allowed by the API and marks it.
	esc, err := svc.Alerts.Escalate(alertID, "sweeper", "")
	if err != nil {
		t.Fatalf("escalate: %v", err)
	}
	if esc.Status != domain.AlertStatusEscalated {
		t.Fatalf("expected escalated, got %s", esc.Status)
	}
}

func TestEscalateSweeper(t *testing.T) {
	svc, _, cfg := newTestEnv(t)
	cfg.AlertEscalateAfter = time.Hour
	r := ingest(t, svc, "monitor_a1", 90000, 30000, 21.0, 45.0, 20.0)
	if len(r.AlertsCreated) == 0 {
		t.Fatal("expected alert")
	}
	// Force the alert's creation time into the past.
	alertID := r.AlertsCreated[0].ID
	a, _ := svc.Store.Alerts().Get(alertID)
	a.CreatedAt = time.Now().UTC().Add(-2 * time.Hour)
	_, _ = svc.Store.Alerts().Update(a)

	sweeper := NewEscalateSweeper(cfg, svc.Store, svc.Alerts, svc.Audit)
	n, err := sweeper.Sweep(time.Now())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 escalation, got %d", n)
	}
	updated, _ := svc.Store.Alerts().Get(alertID)
	if updated.Status != domain.AlertStatusEscalated {
		t.Fatalf("expected escalated, got %s", updated.Status)
	}
}

func TestZoneSweeperAutoInterlock(t *testing.T) {
	svc, _, cfg := newTestEnv(t)
	cfg.AutoInterlockAfter = 10 * time.Minute
	// Elevate zone_a1.
	r := ingest(t, svc, "monitor_a1", 90000, 30000, 21.0, 45.0, 20.0)
	if r.Zone.Status != domain.ZoneStatusElevated {
		t.Fatalf("expected elevated, got %s", r.Zone.Status)
	}
	// Age the status so the sweeper triggers.
	z, _ := svc.Zones.GetCleanZone("zone_a1")
	z.StatusSince = time.Now().UTC().Add(-30 * time.Minute)
	_, _ = svc.Store.CleanZones().Update(z)

	sweeper := NewZoneSweeper(cfg, svc.Store, svc.Interlock, svc.Audit)
	n, err := sweeper.Sweep(time.Now())
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 auto interlock, got %d", n)
	}
	zA, _ := svc.Zones.GetCleanZone("zone_a1")
	if zA.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("expected interlocked after auto sweep, got %s", zA.Status)
	}
	zB, _ := svc.Zones.GetCleanZone("zone_b1")
	if zB.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("area consistency failed after auto sweep: zone_b1 = %s", zB.Status)
	}
}

func TestOverviewAggregation(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ingest(t, svc, "monitor_a1", 90000, 30000, 21.0, 45.0, 20.0) // elevated + alert
	ov, err := svc.Overview.Build()
	if err != nil {
		t.Fatalf("overview: %v", err)
	}
	if ov.TotalZones != 3 {
		t.Fatalf("expected 3 zones, got %d", ov.TotalZones)
	}
	if ov.TotalMonitors != 6 {
		t.Fatalf("expected 6 monitors, got %d", ov.TotalMonitors)
	}
	if ov.ActiveAlerts < 1 {
		t.Fatalf("expected at least 1 active alert, got %d", ov.ActiveAlerts)
	}
	for _, z := range ov.Zones {
		for _, m := range z.MonitorZones {
			if m.LatestSample == nil {
				t.Fatalf("zone %s monitor %s must have a latest sample", z.CleanZone.ID, m.MonitorZone.ID)
			}
		}
	}
}

func TestAuditTrailCoverage(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	ingest(t, svc, "monitor_a1", 140000, 50000, 21.0, 45.0, 20.0)
	_, _ = svc.Interlock.Restore("zone_a1", "eng", "done", "")
	entries, err := svc.Audit.List(0, domain.AuditInterlockIssue)
	if err != nil {
		t.Fatalf("audit: %v", err)
	}
	if len(entries) < 1 {
		t.Fatal("expected interlock.issue audit entries")
	}
}

func TestInterlockPropagationFromRestoredState(t *testing.T) {
	svc, _, _ := newTestEnv(t)
	// Trigger interlock and restore the whole PA-A area.
	ingest(t, svc, "monitor_a1", 140000, 50000, 21.0, 45.0, 20.0)
	if _, err := svc.Interlock.Restore("zone_a1", "eng", "cleaned", ""); err != nil {
		t.Fatalf("restore: %v", err)
	}
	zB, _ := svc.Zones.GetCleanZone("zone_b1")
	if zB.Status != domain.ZoneStatusRestored {
		t.Fatalf("expected zone_b1 restored, got %s", zB.Status)
	}
	// A new over-limit reading must re-interlock the whole area, including
	// the already-restored sibling zone (rule 6 area consistency).
	r := ingest(t, svc, "monitor_a1", 140000, 50000, 21.0, 45.0, 20.0)
	if !r.InterlockIssued {
		t.Fatal("expected interlock re-issue")
	}
	zA, _ := svc.Zones.GetCleanZone("zone_a1")
	if zA.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("expected zone_a1 interlocked, got %s", zA.Status)
	}
	zB, _ = svc.Zones.GetCleanZone("zone_b1")
	if zB.Status != domain.ZoneStatusInterlocked {
		t.Fatalf("expected zone_b1 interlocked (restored -> interlocked), got %s", zB.Status)
	}
}
