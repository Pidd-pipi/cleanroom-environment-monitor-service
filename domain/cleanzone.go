package domain

import (
	"strings"
	"time"
)

// CleanZone is a monitored cleanroom area (a physical cleanroom zone with a
// defined ISO class and process purpose).
type CleanZone struct {
	// ID is the stable unique identifier.
	ID string `json:"id"`
	// Name is the human readable name, e.g. "Litho Line A".
	Name string `json:"name"`
	// PhysicalArea is the shared physical cleanroom. Zones in the same
	// physical area must interlock together.
	PhysicalArea string `json:"physical_area"`
	// IsoClass is the design ISO cleanliness class.
	IsoClass IsoClass `json:"iso_class"`
	// Process is the semiconductor process purpose.
	Process ProcessType `json:"process"`
	// Status is the current state-machine status.
	Status ZoneStatus `json:"status"`
	// StatusSince is when the current status was entered.
	StatusSince time.Time `json:"status_since"`
	// LastParticleRatio is the latest worst concentration-to-limit ratio.
	LastParticleRatio float64 `json:"last_particle_ratio,omitempty"`
	// CreatedAt / UpdatedAt track record lifecycle.
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// NewCleanZone builds a new clean zone in the normal state with sane
// defaults for timestamps.
func NewCleanZone(id, name, physicalArea string, iso IsoClass, process ProcessType) CleanZone {
	now := time.Now().UTC()
	return CleanZone{
		ID:           id,
		Name:         name,
		PhysicalArea: physicalArea,
		IsoClass:     iso,
		Process:      process,
		Status:       ZoneStatusNormal,
		StatusSince:  now,
		CreatedAt:    now,
		UpdatedAt:    now,
	}
}

// Validate performs structural validation of a clean zone.
func (z *CleanZone) Validate() error {
	if strings.TrimSpace(z.ID) == "" {
		return InvalidInput("clean zone id must not be empty")
	}
	if strings.TrimSpace(z.Name) == "" {
		return InvalidInput("clean zone name must not be empty")
	}
	if strings.TrimSpace(z.PhysicalArea) == "" {
		return InvalidInput("clean zone physical area must not be empty")
	}
	if _, err := ParseIsoClass(string(z.IsoClass)); err != nil {
		return InvalidInput("invalid iso_class: " + string(z.IsoClass))
	}
	if _, err := ParseProcessType(string(z.Process)); err != nil {
		return InvalidInput("invalid process: " + string(z.Process))
	}
	if _, err := ParseZoneStatus(string(z.Status)); err != nil {
		return InvalidInput("invalid status: " + string(z.Status))
	}
	return nil
}

// Touch updates the UpdatedAt timestamp.
func (z *CleanZone) Touch() {
	z.UpdatedAt = time.Now().UTC()
}

// SetStatus moves the zone to the given status, updating StatusSince only
// when the status actually changes.
func (z *CleanZone) SetStatus(next ZoneStatus) bool {
	if z.Status == next {
		return false
	}
	z.Status = next
	z.StatusSince = time.Now().UTC()
	z.UpdatedAt = z.StatusSince
	return true
}

// InInterlockedState reports whether the zone is currently under an active
// ventilation interlock (used by area-wide propagation). Only the interlocked
// status counts: over_limit is the pre-interlock overrun state and must not be
// treated as already interlocked, otherwise a sibling over-limit zone would
// be skipped during area propagation and never receive the interlock command.
func (z *CleanZone) InInterlockedState() bool {
	return z.Status == ZoneStatusInterlocked
}
