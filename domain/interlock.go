package domain

import "time"

// InterlockLog records one interlock action issued against a physical
// cleanroom area.
type InterlockLog struct {
	// ID is the stable unique identifier.
	ID string `json:"id"`
	// CleanZoneID is the clean zone that triggered the interlock.
	CleanZoneID string `json:"clean_zone_id"`
	// PhysicalArea is the affected physical area (whole-area consistency).
	PhysicalArea string `json:"physical_area"`
	// TriggerMonitorZoneID is the monitor zone whose reading triggered it.
	TriggerMonitorZoneID string `json:"trigger_monitor_zone_id"`
	// AffectedZoneIDs lists every clean zone that entered interlocked
	// state as part of the same-area propagation.
	AffectedZoneIDs []string `json:"affected_zone_ids"`
	// Action is the interlock command (ffu_speed_up / fresh_air_increase /
	// exhaust_increase).
	Action InterlockAction `json:"action"`
	// Level is the intensity level (e.g. FFU speed percentage delta).
	Level int `json:"level"`
	// Reason is the machine-readable trigger reason.
	Reason string `json:"reason"`
	// PeakRatio is the worst concentration ratio that triggered the event.
	PeakRatio float64 `json:"peak_ratio"`
	// IssuedAt is when the interlock was issued.
	IssuedAt time.Time `json:"issued_at"`
	// RestoredAt is when the restore confirmation closed the event.
	RestoredAt *time.Time `json:"restored_at,omitempty"`
	// RestoreBy is the operator who confirmed the restore.
	RestoreBy string `json:"restore_by,omitempty"`
	// RestoreNote is the operator note for the restore.
	RestoreNote string `json:"restore_note,omitempty"`
}

// NewInterlockLog builds an interlock log entry.
func NewInterlockLog(id, cleanZoneID, physicalArea, triggerZoneID string, affected []string, action InterlockAction, level int, reason string, peakRatio float64) InterlockLog {
	return InterlockLog{
		ID:                   id,
		CleanZoneID:          cleanZoneID,
		PhysicalArea:         physicalArea,
		TriggerMonitorZoneID: triggerZoneID,
		AffectedZoneIDs:      affected,
		Action:               action,
		Level:                level,
		Reason:               reason,
		PeakRatio:            peakRatio,
		IssuedAt:             time.Now().UTC(),
	}
}

// IsOpen reports whether the interlock event is still active.
func (l *InterlockLog) IsOpen() bool {
	return l.RestoredAt == nil
}

// Close marks the interlock event restored.
func (l *InterlockLog) Close(by, note string, at time.Time) {
	l.RestoreBy = by
	l.RestoreNote = note
	t := at.UTC()
	l.RestoredAt = &t
}

// InterlockLevelForRatio maps a concentration ratio to an FFU speed level:
// the higher the ratio, the stronger the ventilation command.
func InterlockLevelForRatio(ratio float64) int {
	switch {
	case ratio >= 3.0:
		return 3
	case ratio >= 2.0:
		return 2
	default:
		return 1
	}
}

// ActionsForLevel returns the command set issued for a given level.
func ActionsForLevel(level int) []InterlockAction {
	actions := []InterlockAction{InterlockFFUSpeedUp, InterlockFreshAirIncrease}
	if level >= 2 {
		actions = append(actions, InterlockExhaustIncrease)
	}
	return actions
}
