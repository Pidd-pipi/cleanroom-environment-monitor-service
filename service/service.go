// Package service implements the application services: sample ingestion,
// interlock orchestration, alert lifecycle, escalation sweepers, audit and
// overview aggregation. Services depend on the store repository interfaces
// and on config, never on HTTP.
package service

import (
	"context"
	"log/slog"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/store"
)

// Services is the application service container.
type Services struct {
	Cfg       *config.Config
	Store     *store.Store
	Ingest    *IngestService
	Interlock *InterlockService
	Alerts    *AlertService
	Audit     *AuditService
	Overview  *OverviewService
	Zones     *ZoneService
}

// New wires every service on top of the shared store and config.
func New(cfg *config.Config, st *store.Store) *Services {
	audit := NewAuditService(st)
	alerts := NewAlertService(cfg, st, audit)
	interlock := NewInterlockService(cfg, st, audit)
	ingest := NewIngestService(cfg, st, alerts, interlock, audit)
	zones := NewZoneService(st, audit)
	overview := NewOverviewService(cfg, st)
	return &Services{
		Cfg:       cfg,
		Store:     st,
		Ingest:    ingest,
		Interlock: interlock,
		Alerts:    alerts,
		Audit:     audit,
		Overview:  overview,
		Zones:     zones,
	}
}

// StartSweepers launches the background escalators. Both stop when the
// context is cancelled.
func (s *Services) StartSweepers(ctx context.Context) {
	es := NewEscalateSweeper(s.Cfg, s.Store, s.Alerts, s.Audit)
	zs := NewZoneSweeper(s.Cfg, s.Store, s.Interlock, s.Audit)
	go es.Run(ctx)
	go zs.Run(ctx)
	slog.Info("service: background sweepers started")
}
