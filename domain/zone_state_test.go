package domain

import (
	"testing"
	"time"
)

func TestCanTransitionTable(t *testing.T) {
	allowed := []struct {
		from, to ZoneStatus
	}{
		{ZoneStatusNormal, ZoneStatusElevated},
		{ZoneStatusNormal, ZoneStatusOverLimit},
		{ZoneStatusNormal, ZoneStatusInterlocked},
		{ZoneStatusElevated, ZoneStatusNormal},
		{ZoneStatusElevated, ZoneStatusOverLimit},
		{ZoneStatusElevated, ZoneStatusInterlocked},
		{ZoneStatusOverLimit, ZoneStatusInterlocked},
		{ZoneStatusOverLimit, ZoneStatusNormal},
		{ZoneStatusInterlocked, ZoneStatusRestored},
		{ZoneStatusRestored, ZoneStatusNormal},
		{ZoneStatusRestored, ZoneStatusElevated},
	}
	for _, tc := range allowed {
		if !CanTransition(tc.from, tc.to) {
			t.Fatalf("expected %s -> %s to be allowed", tc.from, tc.to)
		}
	}
	denied := []struct {
		from, to ZoneStatus
	}{
		{ZoneStatusNormal, ZoneStatusRestored},
		{ZoneStatusElevated, ZoneStatusRestored},
		{ZoneStatusInterlocked, ZoneStatusNormal},
		{ZoneStatusInterlocked, ZoneStatusElevated},
		{ZoneStatusInterlocked, ZoneStatusOverLimit},
		{ZoneStatusOverLimit, ZoneStatusRestored},
	}
	for _, tc := range denied {
		if CanTransition(tc.from, tc.to) {
			t.Fatalf("expected %s -> %s to be denied", tc.from, tc.to)
		}
	}
	// Same state is always allowed.
	if !CanTransition(ZoneStatusInterlocked, ZoneStatusInterlocked) {
		t.Fatal("same-state transition must be allowed")
	}
}

func TestTargetStatusFromRatio(t *testing.T) {
	if got := TargetStatusFromRatio(ZoneStatusNormal, 0.4, 1.5); got != ZoneStatusNormal {
		t.Fatalf("expected normal, got %s", got)
	}
	if got := TargetStatusFromRatio(ZoneStatusNormal, 1.2, 1.5); got != ZoneStatusElevated {
		t.Fatalf("expected elevated, got %s", got)
	}
	if got := TargetStatusFromRatio(ZoneStatusElevated, 1.8, 1.5); got != ZoneStatusOverLimit {
		t.Fatalf("expected over_limit, got %s", got)
	}
	// Interlocked zones do not auto-move back to normal from the ratio.
	if got := TargetStatusFromRatio(ZoneStatusInterlocked, 0.3, 1.5); got != ZoneStatusInterlocked {
		t.Fatalf("expected interlocked stays, got %s", got)
	}
}

func TestStatusAge(t *testing.T) {
	now := time.Now()
	since := now.Add(-5 * time.Minute)
	if got := StatusAge(since, now); got != 5*time.Minute {
		t.Fatalf("expected 5m, got %v", got)
	}
}

func TestAllowedTransitionsFrom(t *testing.T) {
	out := AllowedTransitionsFrom(ZoneStatusNormal)
	if len(out) == 0 {
		t.Fatal("normal should have allowed transitions")
	}
}
