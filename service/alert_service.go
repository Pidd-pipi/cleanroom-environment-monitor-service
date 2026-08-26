package service

import (
	"fmt"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// AlertService handles alert creation with dedup, acknowledgement, closing
// and escalation.
type AlertService struct {
	cfg   *config.Config
	store *store.Store
	audit *AuditService
}

// NewAlertService builds an alert service.
func NewAlertService(cfg *config.Config, st *store.Store, audit *AuditService) *AlertService {
	return &AlertService{cfg: cfg, store: st, audit: audit}
}

// CreateWithDedup raises an alert unless an open/acknowledged alert of the
// same type on the same monitor zone exists within the dedup window, in
// which case the existing alert is merged (occurrence count bump).
func (s *AlertService) CreateWithDedup(cleanZoneID, monitorZoneID string, t domain.AlertType, level domain.AlertLevel, message, sampleID, requestID string) (domain.CleanAlert, bool, error) {
	existing, found, err := s.store.Alerts().FindOpenByDedupKey(domain.DedupKeyOf(monitorZoneID, t))
	if err != nil {
		return domain.CleanAlert{}, false, err
	}
	now := time.Now().UTC()
	if found && now.Sub(existing.CreatedAt) < s.cfg.AlertDedupWindow {
		merged := domain.NewAlert(
			s.store.NewID("alert"), cleanZoneID, monitorZoneID, t, level, message, sampleID, now,
		)
		existing.Merge(merged)
		updated, err := s.store.Alerts().Update(existing)
		if err != nil {
			return domain.CleanAlert{}, false, err
		}
		return updated, false, nil
	}
	alert := domain.NewAlert(
		s.store.NewID("alert"), cleanZoneID, monitorZoneID, t, level, message, sampleID, now,
	)
	created, err := s.store.Alerts().Create(alert)
	if err != nil {
		return domain.CleanAlert{}, false, err
	}
	return created, true, nil
}

// Ack confirms an alert. Operator and disposition are mandatory.
func (s *AlertService) Ack(id, operator, disposition, requestID string) (domain.CleanAlert, error) {
	alert, err := s.store.Alerts().Get(id)
	if err != nil {
		return domain.CleanAlert{}, err
	}
	if err := alert.Ack(operator, disposition, time.Now().UTC()); err != nil {
		return domain.CleanAlert{}, err
	}
	updated, err := s.store.Alerts().Update(alert)
	if err != nil {
		return domain.CleanAlert{}, err
	}
	_ = s.audit.Log(domain.AuditAlertAck, operator, "alert", id,
		fmt.Sprintf("alert %s acknowledged: %s", id, disposition), requestID)
	return updated, nil
}

// Escalate escalates an open alert (used by the sweeper and by manual API).
func (s *AlertService) Escalate(id, operator, requestID string) (domain.CleanAlert, error) {
	alert, err := s.store.Alerts().Get(id)
	if err != nil {
		return domain.CleanAlert{}, err
	}
	alert.Escalate(time.Now().UTC())
	updated, err := s.store.Alerts().Update(alert)
	if err != nil {
		return domain.CleanAlert{}, err
	}
	_ = s.audit.Log(domain.AuditAlertEscalate, operator, "alert", id, "alert escalated for overdue acknowledgement", requestID)
	return updated, nil
}

// Close resolves an alert when the underlying condition clears.
func (s *AlertService) Close(id, operator, reason, requestID string) (domain.CleanAlert, error) {
	alert, err := s.store.Alerts().Get(id)
	if err != nil {
		return domain.CleanAlert{}, err
	}
	alert.Close(time.Now().UTC())
	updated, err := s.store.Alerts().Update(alert)
	if err != nil {
		return domain.CleanAlert{}, err
	}
	_ = s.audit.Log(domain.AuditAlertClose, operator, "alert", id, reason, requestID)
	return updated, nil
}

// ResolveActive closes acknowledged alerts of the given monitor zone and
// type once the underlying condition has cleared. Open and escalated alerts
// stay visible until an engineer acknowledges them (so the 1-hour
// escalation rule keeps working even after the condition clears).
func (s *AlertService) ResolveActive(monitorZoneID string, t domain.AlertType, reason, requestID string) (int, error) {
	all, err := s.store.Alerts().List(store.AlertFilter{MonitorZoneID: monitorZoneID, Type: t})
	if err != nil {
		return 0, err
	}
	closed := 0
	for _, a := range all {
		if a.Status != domain.AlertStatusAcknowledged {
			continue
		}
		a.Close(time.Now().UTC())
		if _, err := s.store.Alerts().Update(a); err != nil {
			return closed, err
		}
		_ = s.audit.Log(domain.AuditAlertClose, "system", "alert", a.ID, reason, requestID)
		closed++
	}
	return closed, nil
}

// List returns alerts with an optional filter.
func (s *AlertService) List(filter store.AlertFilter) ([]domain.CleanAlert, error) {
	return s.store.Alerts().List(filter)
}
