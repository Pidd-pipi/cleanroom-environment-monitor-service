package domain

import "testing"

// TestTargetStatusFromRatioKeepsInterlocked verifies an interlocked zone is
// never pushed to over_limit by a high particle ratio.
func TestTargetStatusFromRatioKeepsInterlocked(t *testing.T) {
	got := TargetStatusFromRatio(ZoneStatusInterlocked, 2.0, 1.5)
	if got != ZoneStatusInterlocked {
		t.Fatalf("interlocked zone must stay interlocked, got %s", got)
	}
}

// TestInInterlockedStateExcludesOverLimit verifies only zones actually in
// the interlocked state are treated as interlocked for area propagation.
func TestInInterlockedStateExcludesOverLimit(t *testing.T) {
	z := NewCleanZone("z1", "Zone 1", "PA-A", Iso5, ProcessLithography)
	z.Status = ZoneStatusOverLimit
	if z.InInterlockedState() {
		t.Fatal("over_limit zone must not be treated as interlocked")
	}
}
