package domain

import (
	"testing"
	"time"
)

// TestRecentSamplesReturnsNewest verifies the recent window keeps the
// newest readings, never the oldest ones.
func TestRecentSamplesReturnsNewest(t *testing.T) {
	base := time.Now().UTC()
	s1 := NewSample("s1", "m1", 100, 50, 21, 45, 18, base.Add(-3*time.Minute))
	s2 := NewSample("s2", "m1", 100, 50, 21, 45, 18, base.Add(-2*time.Minute))
	s3 := NewSample("s3", "m1", 100, 50, 21, 45, 18, base.Add(-time.Minute))
	// newest-first input
	out := RecentSamples([]EnvSample{s3, s2, s1}, 2)
	if len(out) != 2 {
		t.Fatalf("want 2 samples, got %d", len(out))
	}
	if out[0].ID != "s3" || out[1].ID != "s2" {
		t.Fatalf("recent window must keep the newest readings, got [%s %s]", out[0].ID, out[1].ID)
	}
}
