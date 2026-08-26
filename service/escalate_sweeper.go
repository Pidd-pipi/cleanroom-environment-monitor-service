package service

import (
	"context"
	"log/slog"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// EscalateSweeper periodically escalates alerts that stayed unacknowledged
// for longer than the configured window (default 1 hour, sweep every 5
// minutes).
type EscalateSweeper struct {
	cfg     *config.Config
	store   *store.Store
	alerts  *AlertService
	audit   *AuditService
	nowFunc func() time.Time
}

// NewEscalateSweeper builds the escalation sweeper.
func NewEscalateSweeper(cfg *config.Config, st *store.Store, alerts *AlertService, audit *AuditService) *EscalateSweeper {
	return &EscalateSweeper{cfg: cfg, store: st, alerts: alerts, audit: audit, nowFunc: time.Now}
}

// Run blocks until the context is cancelled, escalating overdue alerts on
// every tick.
func (s *EscalateSweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.EscalateSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("service: escalate sweeper stopped")
			return
		case now := <-ticker.C:
			if _, err := s.Sweep(now); err != nil {
				slog.Warn("service: escalate sweeper error", "error", err)
			}
		}
	}
}

// Sweep escalates every alert that has been open for longer than the
// escalation window. Returns the number of escalated alerts.
func (s *EscalateSweeper) Sweep(now time.Time) (int, error) {
	overdue, err := s.store.Alerts().ListOverdue(now, s.cfg.AlertEscalateAfter)
	if err != nil {
		return 0, err
	}
	escalated := 0
	for _, a := range overdue {
		if _, err := s.alerts.Escalate(a.ID, "escalate_sweeper", ""); err != nil {
			return escalated, err
		}
		escalated++
	}
	if escalated > 0 {
		// Best-effort batch audit: alerts are already persisted/escalated, so
		// a failed audit write must not mask the successful escalation.
		// AuditService.Log records the failure with enough context to
		// reconcile the missing trail entry afterwards.
		_ = s.audit.Log(domain.AuditAlertEscalate, "escalate_sweeper", "alert", "",
			"batch escalation of overdue unacknowledged alerts", "")
	}
	return escalated, nil
}
