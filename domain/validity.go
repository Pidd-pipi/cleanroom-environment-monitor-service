package domain

import "time"

// ValidityResult is the outcome of the data-validity evaluation.
type ValidityResult struct {
	// Valid is true when the reading can be used for classification.
	Valid bool
	// Reason explains the invalidation when Valid is false.
	Reason string
}

// EvaluateValidity applies the data-validity rules to a monitor zone's
// equipment: readings from counters in PM maintenance or with expired
// calibration are invalid and must not take part in ISO classification.
// `now` is injectable so tests can freeze time.
func EvaluateValidity(m *MonitorZone, now time.Time) ValidityResult {
	if m == nil {
		return ValidityResult{Valid: false, Reason: "monitor_zone_missing"}
	}
	if m.IsInMaintenance() {
		return ValidityResult{Valid: false, Reason: "counter_in_pm_maintenance"}
	}
	if m.IsCalibrationExpired(now) {
		return ValidityResult{Valid: false, Reason: "calibration_expired"}
	}
	return ValidityResult{Valid: true, Reason: ""}
}

// EvaluateInvalidRatio computes the invalid-sample share for the last
// `window` samples (newest first) of a monitor zone. The data-credibility
// rule flags the zone when this share exceeds the configured threshold.
func EvaluateInvalidRatio(samples []EnvSample, window int, threshold float64) (ratio float64, exceed bool) {
	if window <= 0 {
		window = 50
	}
	if len(samples) == 0 {
		return 0, false
	}
	recent := RecentSamples(samples, window)
	invalid := 0
	for _, s := range recent {
		if !s.Valid {
			invalid++
		}
	}
	ratio = float64(invalid) / float64(len(recent))
	return ratio, ratio > threshold
}
