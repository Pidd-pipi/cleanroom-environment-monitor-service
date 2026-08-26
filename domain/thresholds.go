package domain

import "fmt"

// ProcessThreshold holds the environment thresholds that differ between
// process areas (lithography / etching / diffusion). Particle limits are
// expressed as a multiplier applied on top of the ISO 14644-1 baseline.
type ProcessThreshold struct {
	// Process is the process area the threshold applies to.
	Process ProcessType

	// ParticleMultiplier scales the ISO baseline limit for the monitor
	// zones of this process (stricter processes use values below 1.0).
	ParticleMultiplier float64

	// Temperature, humidity and pressure-difference ranges.
	TempMin     float64
	TempMax     float64
	HumidityMin float64
	HumidityMax float64
	PressureMin float64
	PressureMax float64
}

// ProcessThresholds returns the default threshold set for every supported
// process area. Lithography is the strictest area, diffusion the loosest.
func ProcessThresholds() []ProcessThreshold {
	return []ProcessThreshold{
		{
			Process:            ProcessLithography,
			ParticleMultiplier: 0.8,
			TempMin:            20.0, TempMax: 22.0,
			HumidityMin: 40.0, HumidityMax: 50.0,
			PressureMin: 15.0, PressureMax: 25.0,
		},
		{
			Process:            ProcessEtching,
			ParticleMultiplier: 1.0,
			TempMin:            20.0, TempMax: 24.0,
			HumidityMin: 40.0, HumidityMax: 60.0,
			PressureMin: 10.0, PressureMax: 25.0,
		},
		{
			Process:            ProcessDiffusion,
			ParticleMultiplier: 1.2,
			TempMin:            21.0, TempMax: 25.0,
			HumidityMin: 35.0, HumidityMax: 55.0,
			PressureMin: 5.0, PressureMax: 20.0,
		},
	}
}

// ProcessDefaultsFor returns the process threshold for the given process.
// It falls back to the etching profile for unknown processes so callers can
// never receive a zero-valued threshold.
func ProcessDefaultsFor(p ProcessType) ProcessThreshold {
	for _, t := range ProcessThresholds() {
		if t.Process == p {
			return t
		}
	}
	return ProcessThresholds()[1]
}

// ValidateProcessThresholds ensures the process table is complete and sane.
func ValidateProcessThresholds(ts []ProcessThreshold) error {
	if len(ts) == 0 {
		return fmt.Errorf("domain: process thresholds must not be empty")
	}
	seen := map[ProcessType]bool{}
	for _, t := range ts {
		if t.ParticleMultiplier <= 0 {
			return fmt.Errorf("domain: particle multiplier for %s must be positive", t.Process)
		}
		if t.TempMin >= t.TempMax || t.HumidityMin >= t.HumidityMax || t.PressureMin >= t.PressureMax {
			return fmt.Errorf("domain: invalid range for process %s", t.Process)
		}
		if seen[t.Process] {
			return fmt.Errorf("domain: duplicate process %s", t.Process)
		}
		seen[t.Process] = true
	}
	for _, p := range AllProcessTypes() {
		if !seen[p] {
			return fmt.Errorf("domain: missing thresholds for process %s", p)
		}
	}
	return nil
}
