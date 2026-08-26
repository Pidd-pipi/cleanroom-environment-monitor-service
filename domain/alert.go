package domain

import (
	"fmt"
	"strings"
	"time"
)

// CleanAlert is a cleanroom alert that must be acknowledged and closed by a
// process engineer.
type CleanAlert struct {
	// ID is the stable unique identifier.
	ID string `json:"id"`
	// CleanZoneID / MonitorZoneID locate the alert.
	CleanZoneID   string `json:"clean_zone_id"`
	MonitorZoneID string `json:"monitor_zone_id"`
	// Type is the alert category (particle/temp_humidity/pressure/data_quality).
	Type AlertType `json:"type"`
	// Level is the severity.
	Level AlertLevel `json:"level"`
	// Status is the lifecycle status.
	Status AlertStatus `json:"status"`
	// Message is the human-readable alert text.
	Message string `json:"message"`
	// SampleID links the alert to the triggering sample when applicable.
	SampleID string `json:"sample_id,omitempty"`
	// Count is how many merged occurrences this alert represents.
	Count int `json:"count"`
	// DedupKey is monitor-zone+type used for the dedup window merge.
	DedupKey string `json:"dedup_key"`
	// FirstSeenAt is when the condition first appeared.
	FirstSeenAt time.Time `json:"first_seen_at"`
	// CreatedAt is when the alert record was created/last merged.
	CreatedAt time.Time `json:"created_at"`
	// AcknowledgedAt is when an engineer confirmed the alert.
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	// AckBy is the engineer who acknowledged the alert.
	AckBy string `json:"ack_by,omitempty"`
	// Disposition is the required disposition note.
	Disposition string `json:"disposition,omitempty"`
	// EscalatedAt is when the overdue sweeper escalated the alert.
	EscalatedAt *time.Time `json:"escalated_at,omitempty"`
	// ClosedAt is when the condition resolved.
	ClosedAt *time.Time `json:"closed_at,omitempty"`
}

// DedupKeyOf builds the dedup key for a monitor zone + alert type.
func DedupKeyOf(monitorZoneID string, t AlertType) string {
	return monitorZoneID + ":" + string(t)
}

// NewAlert builds an open alert.
func NewAlert(id, cleanZoneID, monitorZoneID string, t AlertType, level AlertLevel, message, sampleID string, at time.Time) CleanAlert {
	return CleanAlert{
		ID:            id,
		CleanZoneID:   cleanZoneID,
		MonitorZoneID: monitorZoneID,
		Type:          t,
		Level:         level,
		Status:        AlertStatusOpen,
		Message:       message,
		SampleID:      sampleID,
		Count:         1,
		DedupKey:      DedupKeyOf(monitorZoneID, t),
		FirstSeenAt:   at.UTC(),
		CreatedAt:     at.UTC(),
	}
}

// Merge folds a repeated occurrence into an existing alert within the dedup
// window, bumping the counter and refreshing the timestamp/message.
func (a *CleanAlert) Merge(other CleanAlert) {
	a.Count = other.Count
	a.CreatedAt = other.CreatedAt
	a.Message = other.Message
	a.SampleID = other.SampleID
}

// Ack acknowledges the alert with operator + disposition. Disposition is
// mandatory and the alert must not already be closed.
func (a *CleanAlert) Ack(by, disposition string, at time.Time) error {
	if strings.TrimSpace(disposition) == "" {
		return InvalidInput("disposition is required when acknowledging an alert")
	}
	if a.Status == AlertStatusClosed {
		return Conflict(fmt.Sprintf("alert %s is already closed", a.ID))
	}
	a.Status = AlertStatusAcknowledged
	a.AckBy = by
	a.Disposition = disposition
	t := at.UTC()
	a.AcknowledgedAt = &t
	return nil
}

// Escalate escalates an open alert.
func (a *CleanAlert) Escalate(at time.Time) {
	if a.Status == AlertStatusClosed {
		return
	}
	a.Status = AlertStatusEscalated
	t := at.UTC()
	a.EscalatedAt = &t
}

// Close resolves the alert.
func (a *CleanAlert) Close(at time.Time) {
	a.Status = AlertStatusClosed
	t := at.UTC()
	a.ClosedAt = &t
}

// IsActive reports whether the alert still needs attention.
func (a *CleanAlert) IsActive() bool {
	return a.Status == AlertStatusOpen || a.Status == AlertStatusEscalated
}

// NeedsEscalation reports whether the alert exceeded the escalation window
// without being acknowledged.
func (a *CleanAlert) NeedsEscalation(now time.Time, window time.Duration) bool {
	if a.Status != AlertStatusOpen {
		return false
	}
	return now.Sub(a.CreatedAt) >= window
}
