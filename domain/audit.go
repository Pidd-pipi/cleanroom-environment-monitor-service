package domain

import "time"

// AuditAction is the stable action identifier of an audit event.
type AuditAction string

const (
	// AuditSampleIngest is logged whenever a sample is processed.
	AuditSampleIngest AuditAction = "sample.ingest"
	// AuditMaintenanceStart / End are logged when PM is toggled.
	AuditMaintenanceStart AuditAction = "maintenance.start"
	AuditMaintenanceEnd   AuditAction = "maintenance.end"
	// AuditInterlockIssue / Restore are logged for interlock lifecycle.
	AuditInterlockIssue   AuditAction = "interlock.issue"
	AuditInterlockRestore AuditAction = "interlock.restore"
	// AuditAlertAck / Escalate / Close are logged for alert lifecycle.
	AuditAlertAck      AuditAction = "alert.ack"
	AuditAlertEscalate AuditAction = "alert.escalate"
	AuditAlertClose    AuditAction = "alert.close"
	// AuditZoneRestore is logged for clean-zone restore confirmation.
	AuditZoneRestore AuditAction = "zone.restore"
	// AuditHTTPRequest is logged by the audit middleware for every request.
	AuditHTTPRequest AuditAction = "http.request"
	// AuditZoneCreate / MonitorCreate are logged by bootstrap/API creates.
	AuditZoneCreate    AuditAction = "zone.create"
	AuditMonitorCreate AuditAction = "monitor.create"
)

// AuditEntry is one immutable audit record.
type AuditEntry struct {
	// ID is the stable unique identifier.
	ID string `json:"id"`
	// Action is the audit action type.
	Action AuditAction `json:"action"`
	// Operator is the acting user/process.
	Operator string `json:"operator"`
	// TargetType is the entity type (clean_zone/monitor_zone/sample/...).
	TargetType string `json:"target_type"`
	// TargetID is the entity id.
	TargetID string `json:"target_id"`
	// Detail is a free-form description.
	Detail string `json:"detail"`
	// RequestID links the event to the HTTP request trace.
	RequestID string `json:"request_id,omitempty"`
	// CreatedAt is the event time.
	CreatedAt time.Time `json:"created_at"`
}

// NewAuditEntry builds an audit record.
func NewAuditEntry(id string, action AuditAction, operator, targetType, targetID, detail string, at time.Time) AuditEntry {
	return AuditEntry{
		ID:         id,
		Action:     action,
		Operator:   operator,
		TargetType: targetType,
		TargetID:   targetID,
		Detail:     detail,
		CreatedAt:  at.UTC(),
	}
}
