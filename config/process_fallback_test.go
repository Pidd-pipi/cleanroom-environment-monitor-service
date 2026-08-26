package config

import (
	"testing"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// TestForProcessReturnsOwnProfile verifies each process area receives its
// own threshold profile, never another area's profile.
func TestForProcessReturnsOwnProfile(t *testing.T) {
	for _, p := range []domain.ProcessType{domain.ProcessLithography, domain.ProcessEtching, domain.ProcessDiffusion} {
		got := ForProcess(p)
		want := domain.ProcessDefaultsFor(p)
		if got.ParticleMultiplier != want.ParticleMultiplier {
			t.Fatalf("%s must use its own particle multiplier %v, got %v", p, want.ParticleMultiplier, got.ParticleMultiplier)
		}
		if got.TempMin != want.TempMin {
			t.Fatalf("%s must use its own temperature floor %v, got %v", p, want.TempMin, got.TempMin)
		}
	}
}
