package service

import (
	"errors"
	"fmt"
	"log/slog"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// Bootstrap seeds the store with a representative cleanroom topology when
// the store is empty: two physical areas, three clean zones and six monitor
// zones with particle counters.
type Bootstrap struct {
	cfg    *config.Config
	store  *store.Store
	ingest *IngestService
}

// NewBootstrap builds the bootstrap helper.
func NewBootstrap(cfg *config.Config, st *store.Store, ingest *IngestService) *Bootstrap {
	return &Bootstrap{cfg: cfg, store: st, ingest: ingest}
}

// SeedIfEmpty seeds the store when no clean zones exist yet. It is idempotent
// so restarts (or a partially-seeded store after a crash) never duplicate
// fixtures or fail with a conflict: an already-populated store is a no-op,
// and individual fixtures that already exist are skipped rather than treated
// as errors.
func (b *Bootstrap) SeedIfEmpty() error {
	zoneCount, err := b.store.CleanZones().Count()
	if err != nil {
		return fmt.Errorf("bootstrap: count zones: %w", err)
	}
	if zoneCount > 0 {
		// The store was restored from a snapshot or already seeded: do not
		// recreate fixtures. This is the steady-state restart path.
		slog.Info("bootstrap: store already seeded, skipping")
		return nil
	}
	if err := b.seed(); err != nil {
		return fmt.Errorf("bootstrap: %w", err)
	}
	slog.Info("bootstrap: seeded demo cleanroom topology")
	return nil
}

func (b *Bootstrap) seed() error {
	iso := config.ISO14644Limits()
	if err := domain.ValidateLimitTable(iso); err != nil {
		return err
	}
	if err := config.ValidateProcessThresholds(config.ProcessThresholds()); err != nil {
		return err
	}

	// Physical area PA-A: lithography + etching.
	zoneA := domain.NewCleanZone("zone_a1", "Litho Line A", "PA-A", domain.Iso5, domain.ProcessLithography)
	zoneB := domain.NewCleanZone("zone_b1", "Etch Bay B", "PA-A", domain.Iso6, domain.ProcessEtching)
	// Physical area PA-B: diffusion.
	zoneC := domain.NewCleanZone("zone_c1", "Diffusion Hall C", "PA-B", domain.Iso7, domain.ProcessDiffusion)

	for _, z := range []domain.CleanZone{zoneA, zoneB, zoneC} {
		if _, err := b.store.CleanZones().Create(z); err != nil {
			// A concurrent seeder (or a partial seed after a crash) may have
			// already inserted this fixture. Skip it instead of aborting.
			if !isAlreadyExists(err) {
				return err
			}
		}
	}

	mzA1 := domain.NewMonitorZone("monitor_a1", zoneA.ID, "Litho A - Point 1", "PC-1001")
	mzA2 := domain.NewMonitorZone("monitor_a2", zoneA.ID, "Litho A - Point 2", "PC-1002")
	mzB1 := domain.NewMonitorZone("monitor_b1", zoneB.ID, "Etch B - Point 1", "PC-2001")
	mzB2 := domain.NewMonitorZone("monitor_b2", zoneB.ID, "Etch B - Point 2", "PC-2002")
	mzC1 := domain.NewMonitorZone("monitor_c1", zoneC.ID, "Diffusion C - Point 1", "PC-3001")
	mzC2 := domain.NewMonitorZone("monitor_c2", zoneC.ID, "Diffusion C - Point 2", "PC-3002")

	// Give the litho counter an FFU baseline and realistic calibration.
	mzA1.Equipment.CalibrationDue = time.Now().UTC().AddDate(1, 0, 0)
	mzA1.Equipment.FFULevel = 60
	mzA1.Equipment.FreshAirRatio = 30
	mzA2.Equipment.FFULevel = 60
	mzA2.Equipment.FreshAirRatio = 30

	// Seed monitor zones sequentially. The store serialises every mutation
	// behind its write lock (each Create flushes to disk), so spawning a
	// goroutine per monitor added contention and a leaking error channel
	// without any parallelism benefit. Doing it inline is simpler and the
	// per-monitor errors are all checked.
	for _, m := range []domain.MonitorZone{mzA1, mzA2, mzB1, mzB2, mzC1, mzC2} {
		if _, err := b.store.MonitorZones().Create(m); err != nil {
			if !isAlreadyExists(err) {
				return err
			}
		}
	}

	// Seed one clean baseline sample per monitor zone so the overview and
	// detail pages render immediately.
	baseline := []struct {
		monitorID string
		c0303     float64
		c0505     float64
		temp      float64
		hum       float64
		press     float64
	}{
		{mzA1.ID, 30000, 8000, 21.0, 45.0, 20.0},
		{mzA2.ID, 28000, 7500, 21.2, 44.5, 19.5},
		{mzB1.ID, 180000, 60000, 22.0, 50.0, 18.0},
		{mzB2.ID, 170000, 58000, 22.3, 51.0, 17.5},
		{mzC1.ID, 2400000, 900000, 23.0, 45.0, 12.0},
		{mzC2.ID, 2300000, 880000, 23.2, 46.0, 11.5},
	}
	for _, s := range baseline {
		req := IngestRequest{
			MonitorZoneID: s.monitorID,
			Count0303:     s.c0303,
			Count0505:     s.c0505,
			Temperature:   s.temp,
			Humidity:      s.hum,
			PressureDiff:  s.press,
			Timestamp:     time.Now().UTC().Add(-time.Minute),
			Operator:      "bootstrap",
		}
		if _, err := b.ingestBaseline(req); err != nil {
			// A duplicate baseline sample is harmless (idempotent seed).
			if !isAlreadyExists(err) {
				return err
			}
		}
	}
	return nil
}

// isAlreadyExists reports whether err is a conflict indicating that the entity
// already exists. Seed fixtures share stable ids, so a conflict during a
// (re-)seed means the fixture is already present and the seeder can skip it.
func isAlreadyExists(err error) bool {
	var de *domain.Error
	if errors.As(err, &de) {
		return de.Code == domain.CodeConflict
	}
	return false
}

// ingestBaseline reuses the ingest pipeline so seeded samples follow the
// exact same validity/classification rules as live data.
func (b *Bootstrap) ingestBaseline(req IngestRequest) (IngestResult, error) {
	return b.ingest.Process(req)
}
