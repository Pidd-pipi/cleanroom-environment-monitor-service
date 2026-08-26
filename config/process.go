package config

import "example.com/cleanroom-environment-monitor-service/domain"

// ProcessThresholds re-exports the domain process threshold table so the
// config layer stays the single place where threshold configuration is
// consumed and validated.
func ProcessThresholds() []domain.ProcessThreshold {
	return domain.ProcessThresholds()
}

// ForProcess returns the process threshold for the given process.
func ForProcess(p domain.ProcessType) domain.ProcessThreshold {
	return domain.ProcessDefaultsFor(p)
}

// ValidateProcessThresholds delegates to the domain validator.
func ValidateProcessThresholds(ts []domain.ProcessThreshold) error {
	return domain.ValidateProcessThresholds(ts)
}
