package domain

import (
	"math"
	"time"
)

// EnvSample is one environment reading reported by a particle counter on a
// monitor zone.
type EnvSample struct {
	// ID is the stable unique identifier.
	ID string `json:"id"`
	// MonitorZoneID links the sample to a monitor zone.
	MonitorZoneID string `json:"monitor_zone_id"`
	// Count0303 is the particle concentration for >= 0.3um (per m3).
	Count0303 float64 `json:"count_0_3um"`
	// Count0505 is the particle concentration for >= 0.5um (per m3).
	Count0505 float64 `json:"count_0_5um"`
	// Temperature in Celsius.
	Temperature float64 `json:"temperature"`
	// Humidity in percent relative humidity.
	Humidity float64 `json:"humidity"`
	// PressureDiff is the room pressure difference in Pascal.
	PressureDiff float64 `json:"pressure_diff"`
	// Valid is false when the equipment was in PM or calibration expired
	// at the time of the reading.
	Valid bool `json:"valid"`
	// InvalidReason explains why the sample was marked invalid.
	InvalidReason string `json:"invalid_reason,omitempty"`
	// IsoClass is the judged ISO class when the sample is valid.
	IsoClass IsoClass `json:"iso_class,omitempty"`
	// OverTable is true when the concentration exceeded every table entry.
	OverTable bool `json:"over_table,omitempty"`
	// Timestamp is the reading time (report time from the device).
	Timestamp time.Time `json:"timestamp"`
	// ReceivedAt is when the service ingested the sample.
	ReceivedAt time.Time `json:"received_at"`
}

// NewSample builds an empty sample with sane timestamps.
func NewSample(id, monitorZoneID string, count0303, count0505, temp, humidity, pressure float64, ts time.Time) EnvSample {
	now := time.Now().UTC()
	if ts.IsZero() {
		ts = now
	}
	return EnvSample{
		ID:            id,
		MonitorZoneID: monitorZoneID,
		Count0303:     count0303,
		Count0505:     count0505,
		Temperature:   temp,
		Humidity:      humidity,
		PressureDiff:  pressure,
		Valid:         true,
		Timestamp:     ts.UTC(),
		ReceivedAt:    now,
	}
}

// MarkInvalid sets the sample invalid with a reason.
func (s *EnvSample) MarkInvalid(reason string) {
	s.Valid = false
	s.InvalidReason = reason
	s.IsoClass = ""
}

// MaxRatioAgainst returns the worst concentration-to-limit ratio of this
// sample against a monitor zone threshold.
func (s *EnvSample) MaxRatioAgainst(lim IsoLimit) float64 {
	return RatioAgainst(lim, s.Count0303, s.Count0505)
}

// IsBelowLimit reports whether both particle concentrations are within the
// given limit.
func (s *EnvSample) IsBelowLimit(lim IsoLimit) bool {
	return s.Count0303 <= lim.Count0303 && s.Count0505 <= lim.Count0505
}

// IsOutOfRange reports whether temperature/humidity/pressure fall outside
// the given environment range.
func (s *EnvSample) IsOutOfRange(r EnvRange) (temp, humidity, pressure bool) {
	temp = s.Temperature < r.TempMin || s.Temperature > r.TempMax
	humidity = s.Humidity < r.HumidityMin || s.Humidity > r.HumidityMax
	pressure = s.PressureDiff < r.PressureMin || s.PressureDiff > r.PressureMax
	return
}

// ValidSampleRatio computes the share of valid samples in the slice (0 when
// empty). Used for the data-credibility rule.
func ValidSampleRatio(samples []EnvSample) float64 {
	if len(samples) == 0 {
		return 0
	}
	valid := 0
	for _, s := range samples {
		if s.Valid {
			valid++
		}
	}
	return float64(valid) / float64(len(samples))
}

// RecentSamples returns up to `n` samples ordered newest-first (the caller
// should already pass newest-first data; this just bounds the window).
func RecentSamples(samples []EnvSample, n int) []EnvSample {
	if n <= 0 || len(samples) <= n {
		return samples
	}
	return samples[:n]
}

// ClampTemperatureRange returns the temperature clamped to a sane physical
// range so trend charts never explode on malformed input.
func ClampTemperatureRange(t float64) float64 {
	return math.Round(t*10) / 10
}
