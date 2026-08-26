package store

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

func newRaceStore(t *testing.T) *Store {
	t.Helper()
	st := NewMemoryStore()
	z := domain.NewCleanZone("z1", "Zone 1", "PA-A", domain.Iso5, domain.ProcessLithography)
	if _, err := st.CleanZones().Create(z); err != nil {
		t.Fatal(err)
	}
	m := domain.NewMonitorZone("m1", "z1", "Point 1", "PC-1")
	if _, err := st.MonitorZones().Create(m); err != nil {
		t.Fatal(err)
	}
	return st
}

// TestListReturnsDefensiveCopy guards the List contract: callers must be
// able to mutate the returned slice without corrupting the store.
func TestListReturnsDefensiveCopy(t *testing.T) {
	st := newRaceStore(t)
	s := domain.NewSample("s1", "m1", 100, 50, 21, 45, 18, time.Now().UTC())
	if _, err := st.Samples().Append(s, 100); err != nil {
		t.Fatal(err)
	}
	got, err := st.Samples().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("want 1 sample, got %d", len(got))
	}
	got[0].ID = "mutated"
	after, err := st.Samples().List()
	if err != nil {
		t.Fatal(err)
	}
	if len(after) != 1 || after[0].ID == "mutated" {
		t.Fatalf("List returned a reference into the store instead of a copy")
	}
}

// TestListConcurrentAppend exercises the plain list read against a
// concurrent writer; it must be race-free.
func TestListConcurrentAppend(t *testing.T) {
	st := newRaceStore(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_, _ = st.Samples().List()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			s := domain.NewSample(fmt.Sprintf("s%d", i), "m1", float64(i), float64(i), 21, 45, 18, time.Now().UTC())
			_, _ = st.Samples().Append(s, 500)
		}
	}()
	close(start)
	wg.Wait()
}

// TestListByMonitorZoneConcurrentAppend exercises the filtered read path
// against a concurrent writer; it must be race-free.
func TestListByMonitorZoneConcurrentAppend(t *testing.T) {
	st := newRaceStore(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_, _ = st.Samples().ListByMonitorZone("m1", 20)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			s := domain.NewSample(fmt.Sprintf("s%d", i), "m1", float64(i), float64(i), 21, 45, 18, time.Now().UTC())
			_, _ = st.Samples().Append(s, 500)
		}
	}()
	close(start)
	wg.Wait()
}

// TestRecentByMonitorZoneConcurrentAppend exercises the recent-window read
// path against a concurrent writer; it must be race-free.
func TestRecentByMonitorZoneConcurrentAppend(t *testing.T) {
	st := newRaceStore(t)
	start := make(chan struct{})
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			_, _ = st.Samples().RecentByMonitorZone("m1", 10)
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for i := 0; i < 300; i++ {
			s := domain.NewSample(fmt.Sprintf("s%d", i), "m1", float64(i), float64(i), 21, 45, 18, time.Now().UTC())
			_, _ = st.Samples().Append(s, 500)
		}
	}()
	close(start)
	wg.Wait()
}
