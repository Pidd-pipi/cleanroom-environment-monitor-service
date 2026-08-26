package service

import (
	"runtime"
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/store"
)

// TestSeedIfEmptyIdempotentSequential verifies repeated seeding never
// duplicates fixtures and never errors.
func TestSeedIfEmptyIdempotentSequential(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewMemoryStore()
	svc := New(cfg, st)
	boot := NewBootstrap(cfg, st, svc.Ingest)

	if err := boot.SeedIfEmpty(); err != nil {
		t.Fatalf("first seed: %v", err)
	}
	if err := boot.SeedIfEmpty(); err != nil {
		t.Fatalf("second seed must be idempotent, got: %v", err)
	}
	zones, err := st.CleanZones().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(zones) != 3 {
		t.Fatalf("want exactly 3 seeded clean zones, got %d", len(zones))
	}
	monitors, err := st.MonitorZones().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(monitors) != 6 {
		t.Fatalf("want exactly 6 seeded monitor zones, got %d", len(monitors))
	}
}

// TestSeedLeavesNoBlockedGoroutines verifies seeding does not leak worker
// goroutines.
func TestSeedLeavesNoBlockedGoroutines(t *testing.T) {
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewMemoryStore()
	svc := New(cfg, st)
	boot := NewBootstrap(cfg, st, svc.Ingest)

	before := runtime.NumGoroutine()
	if err := boot.SeedIfEmpty(); err != nil {
		t.Fatal(err)
	}
	time.Sleep(300 * time.Millisecond)
	after := runtime.NumGoroutine()
	if after > before+2 {
		t.Fatalf("seeding leaked goroutines: before=%d after=%d", before, after)
	}
}
