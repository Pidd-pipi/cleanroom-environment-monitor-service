package service

import (
	"context"
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/store"
)

func newSweeperCtxEnv(t *testing.T) (*config.Config, *store.Store, *Services) {
	t.Helper()
	cfg := config.Default()
	cfg.DataFile = ""
	st := store.NewMemoryStore()
	return cfg, st, New(cfg, st)
}

// TestZoneSweeperStopsOnCancel verifies the zone sweeper goroutine exits
// when its context is cancelled.
func TestZoneSweeperStopsOnCancel(t *testing.T) {
	cfg, st, svc := newSweeperCtxEnv(t)
	zs := NewZoneSweeper(cfg, st, svc.Interlock, svc.Audit)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		zs.Run(ctx)
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
		// stopped as expected
	case <-time.After(2 * time.Second):
		t.Fatal("zone sweeper did not stop after context cancellation")
	}
}

// TestEscalateSweeperStopsOnCancel verifies the escalate sweeper goroutine
// exits when its context is cancelled.
func TestEscalateSweeperStopsOnCancel(t *testing.T) {
	cfg, st, svc := newSweeperCtxEnv(t)
	es := NewEscalateSweeper(cfg, st, svc.Alerts, svc.Audit)
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		es.Run(ctx)
	}()
	time.Sleep(30 * time.Millisecond)
	cancel()
	select {
	case <-done:
		// stopped as expected
	case <-time.After(2 * time.Second):
		t.Fatal("escalate sweeper did not stop after context cancellation")
	}
}
