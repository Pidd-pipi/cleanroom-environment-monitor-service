package service

import (
	"context"
	"log/slog"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/store"
)

// ZoneSweeper automatically interlocks clean zones that stayed above their
// particle limit (elevated or over_limit) for longer than the auto-interlock
// timeout. This realises the rule "over-limit for 10 minutes without
// recovery automatically enters interlocked ventilation".
type ZoneSweeper struct {
	cfg       *config.Config
	store     *store.Store
	interlock *InterlockService
	audit     *AuditService
	nowFunc   func() time.Time
}

// NewZoneSweeper builds the zone sweeper.
func NewZoneSweeper(cfg *config.Config, st *store.Store, interlock *InterlockService, audit *AuditService) *ZoneSweeper {
	return &ZoneSweeper{cfg: cfg, store: st, interlock: interlock, audit: audit, nowFunc: time.Now}
}

// Run blocks until the context is cancelled, checking candidates on every
// tick.
func (s *ZoneSweeper) Run(ctx context.Context) {
	ticker := time.NewTicker(s.cfg.InterlockSweepInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			slog.Info("service: zone sweeper stopped")
			return
		case now := <-ticker.C:
			if _, err := s.Sweep(now); err != nil {
				slog.Warn("service: zone sweeper error", "error", err)
			}
		}
	}
}

// Sweep finds clean zones that have stayed elevated/over_limit beyond the
// timeout and issues area-wide interlocks for them. Returns the number of
// zones that triggered an automatic interlock.
func (s *ZoneSweeper) Sweep(now time.Time) (int, error) {
	candidates, err := s.interlock.AutoInterlockCandidates(now)
	if err != nil {
		return 0, err
	}
	triggered := 0
	for _, z := range candidates {
		// Find the first monitor zone of this clean zone to act as the
		// trigger reference in the interlock log.
		monitors, err := s.store.MonitorZones().ListByCleanZone(z.ID)
		if err != nil {
			return triggered, err
		}
		trigger := ""
		if len(monitors) > 0 {
			trigger = monitors[0].ID
		}
		ratio := z.LastParticleRatio
		if ratio < 1.0 {
			ratio = 1.0
		}
		if _, _, err := s.interlock.IssueForArea(z.ID, trigger, "auto_interlock_timeout", "", ratio); err != nil {
			return triggered, err
		}
		triggered++
	}
	return triggered, nil
}
