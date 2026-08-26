// Package domain contains the core entities, enums and business rules of
// the cleanroom environment monitor. It has no dependencies on storage,
// HTTP or the outside world so every rule is testable in isolation.
package domain

import (
	"fmt"
	"strings"
)

// ZoneStatus is the state of a clean zone in the cleanliness state machine.
// normal -> elevated -> over_limit -> interlocked -> restored.
type ZoneStatus string

const (
	// ZoneStatusNormal is the healthy operating state.
	ZoneStatusNormal ZoneStatus = "normal"
	// ZoneStatusElevated means particle concentration exceeded the limit.
	ZoneStatusElevated ZoneStatus = "elevated"
	// ZoneStatusOverLimit means concentration reached the over-limit ratio.
	ZoneStatusOverLimit ZoneStatus = "over_limit"
	// ZoneStatusInterlocked means ventilation interlocks were issued.
	ZoneStatusInterlocked ZoneStatus = "interlocked"
	// ZoneStatusRestored is entered after a manual restore confirmation.
	ZoneStatusRestored ZoneStatus = "restored"
)

// AllZoneStatuses returns every valid zone status for validation/UI use.
func AllZoneStatuses() []ZoneStatus {
	return []ZoneStatus{
		ZoneStatusNormal, ZoneStatusElevated, ZoneStatusOverLimit,
		ZoneStatusInterlocked, ZoneStatusRestored,
	}
}

// ParseZoneStatus converts a string into a ZoneStatus, returning an
// invalid-input domain error for unknown values so callers (and the HTTP
// error mapper) classify it as a 400 bad request.
func ParseZoneStatus(s string) (ZoneStatus, error) {
	zs := ZoneStatus(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllZoneStatuses() {
		if v == zs {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown zone status %q", s))
}

// IsoClass is the ISO 14644-1 cleanroom class of a clean zone.
type IsoClass string

const (
	Iso5 IsoClass = "iso5"
	Iso6 IsoClass = "iso6"
	Iso7 IsoClass = "iso7"
	Iso8 IsoClass = "iso8"
)

// AllIsoClasses returns the supported ISO classes ordered from cleanest to
// dirtiest.
func AllIsoClasses() []IsoClass {
	return []IsoClass{Iso5, Iso6, Iso7, Iso8}
}

// ParseIsoClass converts a string into an IsoClass, returning an
// invalid-input domain error for unknown values.
func ParseIsoClass(s string) (IsoClass, error) {
	ic := IsoClass(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllIsoClasses() {
		if v == ic {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown iso class %q", s))
}

// ProcessType is the semiconductor process area of a clean zone.
type ProcessType string

const (
	ProcessLithography ProcessType = "lithography"
	ProcessEtching     ProcessType = "etching"
	ProcessDiffusion   ProcessType = "diffusion"
)

// AllProcessTypes returns every supported process type.
func AllProcessTypes() []ProcessType {
	return []ProcessType{ProcessLithography, ProcessEtching, ProcessDiffusion}
}

// ParseProcessType converts a string into a ProcessType, returning an
// invalid-input domain error for unknown values.
func ParseProcessType(s string) (ProcessType, error) {
	pt := ProcessType(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllProcessTypes() {
		if v == pt {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown process type %q", s))
}

// AlertType classifies cleanroom alerts.
type AlertType string

const (
	AlertParticle     AlertType = "particle"
	AlertTempHumidity AlertType = "temp_humidity"
	AlertPressure     AlertType = "pressure"
	AlertDataQuality  AlertType = "data_quality"
)

// AllAlertTypes returns every alert type.
func AllAlertTypes() []AlertType {
	return []AlertType{AlertParticle, AlertTempHumidity, AlertPressure, AlertDataQuality}
}

// ParseAlertType converts a string into an AlertType, returning an
// invalid-input domain error for unknown values.
func ParseAlertType(s string) (AlertType, error) {
	at := AlertType(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllAlertTypes() {
		if v == at {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown alert type %q", s))
}

// AlertLevel is the severity of an alert.
type AlertLevel string

const (
	AlertLevelInfo     AlertLevel = "info"
	AlertLevelWarning  AlertLevel = "warning"
	AlertLevelCritical AlertLevel = "critical"
)

// AllAlertLevels returns every alert level.
func AllAlertLevels() []AlertLevel {
	return []AlertLevel{AlertLevelInfo, AlertLevelWarning, AlertLevelCritical}
}

// ParseAlertLevel converts a string into an AlertLevel, returning an
// invalid-input domain error for unknown values.
func ParseAlertLevel(s string) (AlertLevel, error) {
	al := AlertLevel(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllAlertLevels() {
		if v == al {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown alert level %q", s))
}

// AlertStatus is the lifecycle status of an alert: open -> acknowledged
// (-> escalated when overdue) -> closed.
type AlertStatus string

const (
	AlertStatusOpen         AlertStatus = "open"
	AlertStatusAcknowledged AlertStatus = "acknowledged"
	AlertStatusEscalated    AlertStatus = "escalated"
	AlertStatusClosed       AlertStatus = "closed"
)

// AllAlertStatuses returns every alert status.
func AllAlertStatuses() []AlertStatus {
	return []AlertStatus{
		AlertStatusOpen, AlertStatusAcknowledged,
		AlertStatusEscalated, AlertStatusClosed,
	}
}

// ParseAlertStatus converts a string into an AlertStatus, returning an
// invalid-input domain error for unknown values.
func ParseAlertStatus(s string) (AlertStatus, error) {
	as := AlertStatus(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllAlertStatuses() {
		if v == as {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown alert status %q", s))
}

// InterlockAction enumerates the interlock commands that can be issued.
type InterlockAction string

const (
	// InterlockFFUSpeedUp increases the fan-filter-unit rotation speed.
	InterlockFFUSpeedUp InterlockAction = "ffu_speed_up"
	// InterlockFreshAirIncrease raises the fresh-air supply ratio.
	InterlockFreshAirIncrease InterlockAction = "fresh_air_increase"
	// InterlockExhaustIncrease raises the exhaust extraction.
	InterlockExhaustIncrease InterlockAction = "exhaust_increase"
)

// AllInterlockActions returns every interlock action.
func AllInterlockActions() []InterlockAction {
	return []InterlockAction{
		InterlockFFUSpeedUp, InterlockFreshAirIncrease, InterlockExhaustIncrease,
	}
}

// ParseInterlockAction converts a string into an InterlockAction, returning
// an invalid-input domain error for unknown values.
func ParseInterlockAction(s string) (InterlockAction, error) {
	ia := InterlockAction(strings.ToLower(strings.TrimSpace(s)))
	for _, v := range AllInterlockActions() {
		if v == ia {
			return v, nil
		}
	}
	return "", InvalidInput(fmt.Sprintf("domain: unknown interlock action %q", s))
}
