package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

func TestNewIDMonotonic(t *testing.T) {
	s := NewMemoryStore()
	ids := []string{s.NewID("zone"), s.NewID("zone"), s.NewID("zone")}
	if ids[0] == ids[1] || ids[1] == ids[2] {
		t.Fatalf("ids must be unique: %v", ids)
	}
}

func TestCleanZoneCRUD(t *testing.T) {
	s := NewMemoryStore()
	z := domain.NewCleanZone("z1", "Zone One", "PA-A", domain.Iso5, domain.ProcessLithography)
	if _, err := s.CleanZones().Create(z); err != nil {
		t.Fatalf("create: %v", err)
	}
	got, err := s.CleanZones().Get("z1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Name != "Zone One" {
		t.Fatalf("unexpected name %q", got.Name)
	}
	if _, err := s.CleanZones().Create(z); err == nil {
		t.Fatal("duplicate create must fail")
	}
	if _, err := s.CleanZones().Get("nope"); err == nil {
		t.Fatal("missing get must fail")
	}
	z.Status = domain.ZoneStatusElevated
	if _, err := s.CleanZones().Update(z); err != nil {
		t.Fatalf("update: %v", err)
	}
	zones, _ := s.CleanZones().ListByPhysicalArea("PA-A")
	if len(zones) != 1 {
		t.Fatalf("expected 1 zone in PA-A, got %d", len(zones))
	}
	if err := s.CleanZones().Delete("z1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.CleanZones().Get("z1"); err == nil {
		t.Fatal("deleted zone must be gone")
	}
}

func TestMonitorZoneCascadeDelete(t *testing.T) {
	s := NewMemoryStore()
	z := domain.NewCleanZone("z1", "Zone", "PA-A", domain.Iso5, domain.ProcessLithography)
	_, _ = s.CleanZones().Create(z)
	m := domain.NewMonitorZone("m1", "z1", "Point", "PC-1")
	_, _ = s.MonitorZones().Create(m)
	if err := s.CleanZones().Delete("z1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, err := s.MonitorZones().Get("m1"); err == nil {
		t.Fatal("monitor zone must be cascaded on clean zone delete")
	}
}

func TestMonitorZoneRequiresCleanZone(t *testing.T) {
	s := NewMemoryStore()
	m := domain.NewMonitorZone("m1", "missing", "Point", "PC-1")
	if _, err := s.MonitorZones().Create(m); err == nil {
		t.Fatal("monitor zone without clean zone must fail")
	}
}

func TestSampleAppendAndTrim(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 10; i++ {
		sample := domain.NewSample("s", "m1", float64(i), 0, 21, 45, 18, time.Now())
		sample.ID = "s" + itoa(i)
		if _, err := s.Samples().Append(sample, 5); err != nil {
			t.Fatalf("append: %v", err)
		}
	}
	all, _ := s.Samples().List()
	if len(all) != 5 {
		t.Fatalf("expected trimmed to 5, got %d", len(all))
	}
	recent, _ := s.Samples().RecentByMonitorZone("m1", 3)
	if len(recent) != 3 {
		t.Fatalf("expected 3 recent, got %d", len(recent))
	}
}

func TestPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")
	s := NewStore(path)
	z := domain.NewCleanZone("z1", "Zone", "PA-A", domain.Iso5, domain.ProcessLithography)
	_, _ = s.CleanZones().Create(z)

	s2 := NewStore(path)
	if err := s2.Load(); err != nil {
		t.Fatalf("load: %v", err)
	}
	got, err := s2.CleanZones().Get("z1")
	if err != nil {
		t.Fatalf("get after reload: %v", err)
	}
	if got.ID != "z1" {
		t.Fatalf("unexpected reloaded zone %q", got.ID)
	}
}

func TestLoadMissingFileIsNoop(t *testing.T) {
	s := NewStore(filepath.Join(t.TempDir(), "nope.json"))
	if err := s.Load(); err != nil {
		t.Fatalf("load missing file must be a no-op: %v", err)
	}
}

func TestAuditStoreList(t *testing.T) {
	s := NewMemoryStore()
	for i := 0; i < 3; i++ {
		e := domain.NewAuditEntry("a"+itoa(i), domain.AuditSampleIngest, "sys", "sample", "s"+itoa(i), "d", time.Now())
		_, _ = s.Audit().Create(e)
	}
	entries, err := s.Audit().ListByAction(domain.AuditSampleIngest)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}
}

func TestAlertStoreFindOpenByDedupKey(t *testing.T) {
	s := NewMemoryStore()
	a1 := domain.NewAlert("a1", "z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "m", "s1", time.Now())
	_, _ = s.Alerts().Create(a1)
	a2 := domain.NewAlert("a2", "z1", "m1", domain.AlertParticle, domain.AlertLevelWarning, "m", "s2", time.Now().Add(-time.Hour))
	_, _ = s.Alerts().Create(a2)
	best, found, err := s.Alerts().FindOpenByDedupKey("m1:particle")
	if err != nil {
		t.Fatalf("find: %v", err)
	}
	if !found {
		t.Fatal("must find open alert")
	}
	if best.ID != "a1" {
		t.Fatalf("expected newest alert a1, got %s", best.ID)
	}
}

func TestInterlockStoreOpenCount(t *testing.T) {
	s := NewMemoryStore()
	now := time.Now()
	l1 := domain.NewInterlockLog("il1", "z1", "PA-A", "m1", []string{"z1"}, domain.InterlockFFUSpeedUp, 1, "r", 1.6)
	_, _ = s.Interlocks().Create(l1)
	l2 := domain.NewInterlockLog("il2", "z2", "PA-B", "m2", []string{"z2"}, domain.InterlockFFUSpeedUp, 1, "r", 1.7)
	_, _ = s.Interlocks().Create(l2)
	l2.Close("eng", "ok", now)
	_, _ = s.Interlocks().Update(l2)
	n, _ := s.Interlocks().OpenCount()
	if n != 1 {
		t.Fatalf("expected 1 open interlock, got %d", n)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

func TestStoreFileUnusedCleanup(t *testing.T) {
	dir := t.TempDir()
	s := NewStore(filepath.Join(dir, "x.json"))
	if err := s.Save(); err != nil {
		t.Fatalf("save: %v", err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 file, got %d", len(entries))
	}
}
