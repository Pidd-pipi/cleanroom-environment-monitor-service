package domain

import (
	"strings"
	"time"
)

// ZoneThresholds is the environment threshold set of a monitor zone. When
// fields are zero the effective thresholds are derived from the process
// defaults of the owning clean zone.
type ZoneThresholds struct {
	// Limit0303 / Limit0505 override the ISO-derived particle limits
	// (particles/m3). Zero means "derive from ISO + process multiplier".
	Limit0303 float64 `json:"limit_0_3um,omitempty"`
	Limit0505 float64 `json:"limit_0_5um,omitempty"`
	// TempMin/TempMax is the acceptable temperature range in Celsius.
	TempMin float64 `json:"temp_min,omitempty"`
	TempMax float64 `json:"temp_max,omitempty"`
	// HumidityMin/HumidityMax is the acceptable relative humidity range.
	HumidityMin float64 `json:"humidity_min,omitempty"`
	HumidityMax float64 `json:"humidity_max,omitempty"`
	// PressureMin/PressureMax is the acceptable room pressure difference
	// against the corridor (Pascal).
	PressureMin float64 `json:"pressure_min,omitempty"`
	PressureMax float64 `json:"pressure_max,omitempty"`
}

// HasOverrides reports whether any particle limit override is set.
func (t ZoneThresholds) HasOverrides() bool {
	return t.Limit0303 > 0 || t.Limit0505 > 0
}

// EquipmentStatus tracks the particle counter attached to a monitor zone.
type EquipmentStatus struct {
	// CounterModel is the particle counter model.
	CounterModel string `json:"counter_model,omitempty"`
	// CalibrationDue is the calibration expiry date of the counter.
	CalibrationDue time.Time `json:"calibration_due,omitempty"`
	// InMaintenance marks the counter as being in PM maintenance.
	InMaintenance bool `json:"in_maintenance"`
	// MaintenanceNote explains the ongoing maintenance.
	MaintenanceNote string `json:"maintenance_note,omitempty"`
	// MaintenanceSince is when maintenance started.
	MaintenanceSince *time.Time `json:"maintenance_since,omitempty"`
	// FFULevel is the current fan-filter-unit speed level (0-100).
	FFULevel int `json:"ffu_level"`
	// FreshAirRatio is the fresh-air supply ratio percentage.
	FreshAirRatio int `json:"fresh_air_ratio"`
}

// MonitorZone is a single sampling point inside a clean zone.
type MonitorZone struct {
	ID                string          `json:"id"`
	CleanZoneID       string          `json:"clean_zone_id"`
	Name              string          `json:"name"`
	ParticleCounterID string          `json:"particle_counter_id"`
	Thresholds        ZoneThresholds  `json:"thresholds"`
	Equipment         EquipmentStatus `json:"equipment"`
	CreatedAt         time.Time       `json:"created_at"`
	UpdatedAt         time.Time       `json:"updated_at"`
}

// NewMonitorZone builds a monitor zone with sane defaults.
func NewMonitorZone(id, cleanZoneID, name, counterID string) MonitorZone {
	now := time.Now().UTC()
	return MonitorZone{
		ID:                id,
		CleanZoneID:       cleanZoneID,
		Name:              name,
		ParticleCounterID: counterID,
		Equipment: EquipmentStatus{
			CalibrationDue: now.AddDate(1, 0, 0),
			FFULevel:       60,
			FreshAirRatio:  30,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate performs structural validation of a monitor zone.
func (m *MonitorZone) Validate() error {
	if strings.TrimSpace(m.ID) == "" {
		return InvalidInput("monitor zone id must not be empty")
	}
	if strings.TrimSpace(m.CleanZoneID) == "" {
		return InvalidInput("monitor zone clean_zone_id must not be empty")
	}
	if strings.TrimSpace(m.Name) == "" {
		return InvalidInput("monitor zone name must not be empty")
	}
	if strings.TrimSpace(m.ParticleCounterID) == "" {
		return InvalidInput("particle_counter_id must not be empty")
	}
	if m.Equipment.FFULevel < 0 || m.Equipment.FFULevel > 100 {
		return InvalidInput("ffu_level must be within 0..100")
	}
	return nil
}

// IsInMaintenance reports whether the attached counter is in PM.
func (m *MonitorZone) IsInMaintenance() bool {
	return m.Equipment.InMaintenance
}

// IsCalibrationExpired reports whether the counter calibration is overdue.
// A counter whose calibration due date was never set (the zero value, e.g. a
// legacy monitor loaded from an old snapshot) is treated as "not configured"
// rather than expired, so its readings are not silently discarded.
func (m *MonitorZone) IsCalibrationExpired(now time.Time) bool {
	if m.Equipment.CalibrationDue.IsZero() {
		return false
	}
	return now.After(m.Equipment.CalibrationDue)
}

// SetMaintenance toggles the PM maintenance flag on the equipment. Starting
// maintenance stamps MaintenanceSince; ending maintenance clears it so the
// stale timestamp does not linger after the counter is back in service.
func (m *MonitorZone) SetMaintenance(inMaintenance bool, note string) {
	now := time.Now().UTC()
	m.Equipment.InMaintenance = inMaintenance
	m.Equipment.MaintenanceNote = note
	if inMaintenance {
		m.Equipment.MaintenanceSince = &now
	} else {
		m.Equipment.MaintenanceSince = nil
	}
	m.UpdatedAt = now
}

// Touch updates the UpdatedAt timestamp.
func (m *MonitorZone) Touch() {
	m.UpdatedAt = time.Now().UTC()
}

// EffectiveLimits resolves the particle limits of the zone given the ISO
// baseline table and the process threshold. Explicit overrides win.
func (m *MonitorZone) EffectiveLimits(isoClass IsoClass, isoTable map[IsoClass]IsoLimit, multiplier float64) IsoLimit {
	base := isoTable[isoClass]
	if m.Thresholds.Limit0303 > 0 {
		base.Count0303 = m.Thresholds.Limit0303
	} else {
		base.Count0303 *= multiplier
	}
	if m.Thresholds.Limit0505 > 0 {
		base.Count0505 = m.Thresholds.Limit0505
	} else {
		base.Count0505 *= multiplier
	}
	return base
}

// EffectiveEnvRange resolves the environment ranges, falling back to the
// process defaults when no override is set. A monitor zone that never had
// its temperature/humidity/pressure thresholds configured is therefore judged
// against the defaults of its owning process instead of an all-zero range
// (which would flag every reading as out of range).
func (m *MonitorZone) EffectiveEnvRange(process ProcessType) EnvRange {
	def := ProcessDefaultsFor(process)
	out := EnvRange{
		TempMin:     def.TempMin,
		TempMax:     def.TempMax,
		HumidityMin: def.HumidityMin,
		HumidityMax: def.HumidityMax,
		PressureMin: def.PressureMin,
		PressureMax: def.PressureMax,
	}
	if m.Thresholds.TempMin != 0 {
		out.TempMin = m.Thresholds.TempMin
	}
	if m.Thresholds.TempMax != 0 {
		out.TempMax = m.Thresholds.TempMax
	}
	if m.Thresholds.HumidityMin != 0 {
		out.HumidityMin = m.Thresholds.HumidityMin
	}
	if m.Thresholds.HumidityMax != 0 {
		out.HumidityMax = m.Thresholds.HumidityMax
	}
	if m.Thresholds.PressureMin != 0 {
		out.PressureMin = m.Thresholds.PressureMin
	}
	if m.Thresholds.PressureMax != 0 {
		out.PressureMax = m.Thresholds.PressureMax
	}
	return out
}

// EnvRange is a resolved environment threshold range.
type EnvRange struct {
	TempMin, TempMax         float64
	HumidityMin, HumidityMax float64
	PressureMin, PressureMax float64
}
