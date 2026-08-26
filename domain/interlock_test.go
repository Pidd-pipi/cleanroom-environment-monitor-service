package domain

import (
	"testing"
	"time"
)

func TestInterlockLevelForRatio(t *testing.T) {
	cases := []struct {
		ratio float64
		want  int
	}{
		{1.5, 1},
		{1.9, 1},
		{2.0, 2},
		{2.9, 2},
		{3.0, 3},
		{5.0, 3},
	}
	for _, tc := range cases {
		if got := InterlockLevelForRatio(tc.ratio); got != tc.want {
			t.Fatalf("ratio %.1f: expected level %d, got %d", tc.ratio, tc.want, got)
		}
	}
}

func TestActionsForLevel(t *testing.T) {
	l1 := ActionsForLevel(1)
	if len(l1) != 2 {
		t.Fatalf("level 1 must have 2 actions, got %d", len(l1))
	}
	l3 := ActionsForLevel(3)
	if len(l3) != 3 {
		t.Fatalf("level 3 must have 3 actions, got %d", len(l3))
	}
	if l3[2] != InterlockExhaustIncrease {
		t.Fatalf("level 3 must include exhaust increase, got %s", l3[2])
	}
}

func TestInterlockLogLifecycle(t *testing.T) {
	now := time.Now()
	log := NewInterlockLog("il1", "z1", "PA-A", "m1", []string{"z1", "z2"}, InterlockFFUSpeedUp, 2, "over", 2.3)
	if !log.IsOpen() {
		t.Fatal("new log must be open")
	}
	log.Close("eng", "restored", now)
	if log.IsOpen() {
		t.Fatal("closed log must not be open")
	}
	if log.RestoreBy != "eng" {
		t.Fatalf("unexpected restore by %q", log.RestoreBy)
	}
}
