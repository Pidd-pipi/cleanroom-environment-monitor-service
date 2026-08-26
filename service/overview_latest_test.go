package service

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

func newOverviewEnv(t *testing.T) (*config.Config, *store.Store, *OverviewService) {
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
	return cfg, st, NewOverviewService(cfg, st)
}

// TestOverviewLatestSampleNewestPerMonitor verifies the dashboard shows the
// newest sample of each monitor zone as the latest reading.
func TestOverviewLatestSampleNewestPerMonitor(t *testing.T) {
	cfg, st, svc := newOverviewEnv(t)
	base := time.Now().UTC().Add(-3 * time.Minute)
	for i := 0; i < 3; i++ {
		s := domain.NewSample(fmt.Sprintf("s%d", i), "m1", float64(1000*i), float64(500*i), 21, 45, 18, base.Add(time.Duration(i)*time.Minute))
		if _, err := st.Samples().Append(s, 100); err != nil {
			t.Fatal(err)
		}
	}
	ov, err := svc.Build()
	if err != nil {
		t.Fatal(err)
	}
	if len(ov.Zones) != 1 || len(ov.Zones[0].MonitorZones) != 1 {
		t.Fatalf("unexpected overview shape: %d zones", len(ov.Zones))
	}
	ls := ov.Zones[0].MonitorZones[0].LatestSample
	if ls == nil {
		t.Fatal("latest sample must be present")
	}
	want := base.Add(2 * time.Minute)
	if !ls.Timestamp.Equal(want) {
		t.Fatalf("latest sample must be the newest reading, got %s want %s", ls.Timestamp.Format("15:04:05"), want.Format("15:04:05"))
	}
	_ = cfg
}

// TestOverviewBuildConcurrentSampleAppend runs the aggregate build against
// a concurrent sample writer; it must be race-free and stay consistent.
func TestOverviewBuildConcurrentSampleAppend(t *testing.T) {
	_, st, svc := newOverviewEnv(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			if _, err := svc.Build(); err != nil {
				t.Errorf("build: %v", err)
				return
			}
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 200; i++ {
			s := domain.NewSample(fmt.Sprintf("s%d", i), "m1", float64(i), float64(i), 21, 45, 18, time.Now().UTC())
			_, _ = st.Samples().Append(s, 500)
		}
	}()
	close(start)
	wg.Wait()

	ov, err := svc.Build()
	if err != nil {
		t.Fatal(err)
	}
	if ov.TotalMonitors != 1 {
		t.Fatalf("want 1 monitor in overview, got %d", ov.TotalMonitors)
	}
}
