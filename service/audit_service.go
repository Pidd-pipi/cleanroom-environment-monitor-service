package service

import (
	"fmt"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// AuditService records cross-cutting operation audit trails.
type AuditService struct {
	store *store.Store
}

// NewAuditService builds an audit service.
func NewAuditService(st *store.Store) *AuditService {
	return &AuditService{store: st}
}

// Log records an audit entry. Operator falls back to "system" when empty.
func (s *AuditService) Log(action domain.AuditAction, operator, targetType, targetID, detail, requestID string) error {
	if operator == "" {
		operator = "system"
	}
	entry := domain.NewAuditEntry(
		s.store.NewID("audit"),
		action,
		operator,
		targetType,
		targetID,
		detail,
		time.Now().UTC(),
	)
	entry.RequestID = requestID
	_, err := s.store.Audit().Create(entry)
	if err != nil {
		return fmt.Errorf("audit: %w", err)
	}
	return nil
}

// List returns audit entries newest first.
func (s *AuditService) List(limit int, action domain.AuditAction) ([]domain.AuditEntry, error) {
	return s.store.Audit().List(limit, action)
}

// Count returns the total number of audit records.
func (s *AuditService) Count() (int, error) {
	return s.store.Audit().Count()
}
