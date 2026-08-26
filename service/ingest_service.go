package service

import (
	"fmt"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// IngestRequest is the validated input of a sample ingestion.
type IngestRequest struct {
	MonitorZoneID string
	Count0303     float64
	Count0505     float64
	Temperature   float64
	Humidity      float64
	PressureDiff  float64
	Timestamp     time.Time
	Operator      string
	RequestID     string
}

// IngestResult summarises everything that happened while processing one
// sample: the persisted sample, the state transition, interlock and alerts.
type IngestResult struct {
	Sample             domain.EnvSample     `json:"sample"`
	Zone               domain.CleanZone     `json:"zone"`
	MonitorZone        domain.MonitorZone   `json:"monitor_zone"`
	StateChanged       bool                 `json:"state_changed"`
	InterlockIssued    bool                 `json:"interlock_issued"`
	InterlockLog       *domain.InterlockLog `json:"interlock_log,omitempty"`
	AlertsCreated      []domain.CleanAlert  `json:"alerts_created"`
	AlertsMerged       []domain.CleanAlert  `json:"alerts_merged"`
	AlertsClosed       int                  `json:"alerts_closed"`
	ParticleRatio      float64              `json:"particle_ratio"`
	InvalidRatio       float64              `json:"invalid_ratio"`
	DataCredibilityLow bool                 `json:"data_credibility_low"`
}

// IngestService processes environment sample reports end-to-end: validity
// evaluation, ISO classification, state-machine transitions, area-wide
// interlocks and alert generation/dedup.
type IngestService struct {
	cfg       *config.Config
	store     *store.Store
	alerts    *AlertService
	interlock *InterlockService
	audit     *AuditService
}

// NewIngestService builds the ingest service.
func NewIngestService(cfg *config.Config, st *store.Store, alerts *AlertService, interlock *InterlockService, audit *AuditService) *IngestService {
	return &IngestService{cfg: cfg, store: st, alerts: alerts, interlock: interlock, audit: audit}
}

// Process handles one reported sample and returns the full result.
func (s *IngestService) Process(req IngestRequest) (IngestResult, error) {
	res := IngestResult{}
	monitor, err := s.store.MonitorZones().Get(req.MonitorZoneID)
	if err != nil {
		return res, err
	}
	zone, err := s.store.CleanZones().Get(monitor.CleanZoneID)
	if err != nil {
		return res, err
	}
	res.MonitorZone = monitor

	sample := domain.NewSample(
		s.store.NewID("sample"),
		monitor.ID,
		req.Count0303, req.Count0505,
		req.Temperature, req.Humidity, req.PressureDiff,
		req.Timestamp,
	)

	// ---- Data validity (rule 3) -------------------------------------
	ts := sample.Timestamp
	if ts.IsZero() {
		ts = time.Now().UTC()
	}
	validity := domain.EvaluateValidity(nil, ts)
	if !validity.Valid {
		sample.MarkInvalid(validity.Reason)
	}
	res.Sample = sample

	isoTable := config.ISO14644Limits()
	processThreshold := config.ForProcess(domain.ProcessEtching)
	limits := monitor.EffectiveLimits(zone.IsoClass, isoTable, processThreshold.ParticleMultiplier)
	envRange := monitor.EffectiveEnvRange(domain.ProcessEtching)

	if sample.Valid {
		// ---- ISO classification (rule 2) ------------------------------
		iso, overTable := domain.ClassifyISO(isoTable, sample.Count0303, sample.Count0505)
		sample.IsoClass = iso
		sample.OverTable = overTable
		res.ParticleRatio = sample.MaxRatioAgainst(limits)

		// ---- State machine + area interlock (rules 1 & 4) -------------
		changed, interlockIssued, logEntry, err := s.applyStateAndInterlock(&zone, &monitor, sample, res.ParticleRatio, req.RequestID)
		if err != nil {
			return res, err
		}
		res.StateChanged = changed
		res.InterlockIssued = interlockIssued
		res.InterlockLog = logEntry

		// Persist the latest particle ratio on the clean zone so the
		// auto-interlock sweeper can evaluate sustained over-limit zones.
		// Reload the zone first: an area-wide interlock may have moved it
		// to interlocked inside applyStateAndInterlock and the local copy
		// would otherwise overwrite that transition.
		freshZone, err := s.store.CleanZones().Get(zone.ID)
		if err != nil {
			return res, err
		}
		zone = freshZone
		if zone.LastParticleRatio != res.ParticleRatio {
			zone.LastParticleRatio = res.ParticleRatio
			zone.Touch()
			if _, err := s.store.CleanZones().Update(zone); err != nil {
				return res, err
			}
		}

		// ---- Particle alert -------------------------------------------
		if res.ParticleRatio >= 1.0 {
			level := domain.AlertLevelWarning
			if res.ParticleRatio >= s.cfg.OverLimitRatio {
				level = domain.AlertLevelCritical
			}
			msg := fmt.Sprintf("particle concentration above limit: ratio %.2f (0.3um=%.0f/m3, 0.5um=%.0f/m3)",
				res.ParticleRatio, sample.Count0303, sample.Count0505)
			created, isNew, err := s.alerts.CreateWithDedup(zone.ID, monitor.ID, domain.AlertParticle, level, msg, sample.ID, req.RequestID)
			if err == nil {
				res.recordAlert(created, isNew)
			}
		} else {
			n, _ := s.alerts.ResolveActive(monitor.ID, domain.AlertParticle, "particle concentration back within limit", req.RequestID)
			res.AlertsClosed += n
		}

		// ---- Environment alerts (temp/humidity/pressure) --------------
		tempBad, humBad, pressBad := sample.IsOutOfRange(envRange)
		if tempBad || humBad {
			level := domain.AlertLevelWarning
			if tempBad {
				level = domain.AlertLevelCritical
			}
			msg := fmt.Sprintf("temperature/humidity out of range: temp=%.1fC hum=%.1f%%",
				sample.Temperature, sample.Humidity)
			created, isNew, err := s.alerts.CreateWithDedup(zone.ID, monitor.ID, domain.AlertTempHumidity, level, msg, sample.ID, req.RequestID)
			if err == nil {
				res.recordAlert(created, isNew)
			}
		} else {
			n, _ := s.alerts.ResolveActive(monitor.ID, domain.AlertTempHumidity, "temperature/humidity back in range", req.RequestID)
			res.AlertsClosed += n
		}
		if pressBad {
			msg := fmt.Sprintf("pressure difference out of range: %.1f Pa (expected %.1f..%.1f)",
				sample.PressureDiff, envRange.PressureMin, envRange.PressureMax)
			created, isNew, err := s.alerts.CreateWithDedup(zone.ID, monitor.ID, domain.AlertPressure, domain.AlertLevelWarning, msg, sample.ID, req.RequestID)
			if err == nil {
				res.recordAlert(created, isNew)
			}
		} else {
			n, _ := s.alerts.ResolveActive(monitor.ID, domain.AlertPressure, "pressure back in range", req.RequestID)
			res.AlertsClosed += n
		}
	}

	// ---- Data credibility (rule 3: >30% invalid) ----------------------
	recent, err := s.store.Samples().RecentByMonitorZone(monitor.ID, s.cfg.SampleWindow())
	if err != nil {
		return res, err
	}
	window := append([]domain.EnvSample{sample}, recent...)
	if len(window) > s.cfg.SampleWindow() {
		window = window[:s.cfg.SampleWindow()]
	}
	invalidRatio, low := domain.EvaluateInvalidRatio(window, s.cfg.SampleWindow(), s.cfg.InvalidRatioThreshold)
	res.InvalidRatio = invalidRatio
	res.DataCredibilityLow = low
	if low {
		msg := fmt.Sprintf("data credibility low: %.0f%% of recent samples invalid", invalidRatio*100)
		created, isNew, err := s.alerts.CreateWithDedup(zone.ID, monitor.ID, domain.AlertDataQuality, domain.AlertLevelWarning, msg, sample.ID, req.RequestID)
		if err == nil {
			res.recordAlert(created, isNew)
		}
	} else if invalidRatio > 0 {
		n, _ := s.alerts.ResolveActive(monitor.ID, domain.AlertDataQuality, "data credibility restored", req.RequestID)
		res.AlertsClosed += n
	}

	// ---- Persist sample + audit --------------------------------------
	if _, err := s.store.Samples().Append(sample, s.cfg.MaxSamplesPerZone); err != nil {
		return res, err
	}
	_ = s.audit.Log(domain.AuditSampleIngest, req.Operator, "sample", sample.ID,
		fmt.Sprintf("monitor=%s valid=%v ratio=%.3f", monitor.ID, sample.Valid, res.ParticleRatio), req.RequestID)

	// Re-read the zone/monitor so the result reflects all updates.
	if z, err := s.store.CleanZones().Get(zone.ID); err == nil {
		res.Zone = z
	}
	if m, err := s.store.MonitorZones().Get(monitor.ID); err == nil {
		res.MonitorZone = m
	}
	return res, nil
}

// applyStateAndInterlock updates the clean-zone state machine and issues
// area-wide interlocks when the particle ratio crosses the over-limit
// threshold. Returns whether the state changed, whether an interlock was
// issued and the interlock log (nil when none).
func (s *IngestService) applyStateAndInterlock(zone *domain.CleanZone, monitor *domain.MonitorZone, sample domain.EnvSample, ratio float64, requestID string) (bool, bool, *domain.InterlockLog, error) {
	// Rule 4: at >= 1.5x the limit an interlock is issued immediately.
	if ratio >= s.cfg.OverLimitRatio {
		if zone.Status == domain.ZoneStatusInterlocked {
			// The area is already ventilating; no duplicate command.
			return false, false, nil, nil
		}
		logEntry, _, err := s.interlock.IssueForArea(zone.ID, monitor.ID, "particle_over_limit", requestID, ratio)
		if err != nil {
			return false, false, nil, err
		}
		return true, true, &logEntry, nil
	}

	// Recovery below the limit: an interlocked zone stays interlocked
	// until manual restore confirmation (rule 4).
	if ratio < 1.0 && zone.Status == domain.ZoneStatusInterlocked {
		return false, false, nil, nil
	}

	target := domain.TargetStatusFromRatio(zone.Status, ratio, s.cfg.OverLimitRatio)
	if target != zone.Status {
		zone.SetStatus(target)
		if _, err := s.store.CleanZones().Update(*zone); err != nil {
			return false, false, nil, err
		}
		return true, false, nil, nil
	}
	return false, false, nil, nil
}

// recordAlert folds an alert into the result bucket.
func (r *IngestResult) recordAlert(a domain.CleanAlert, isNew bool) {
	if isNew {
		r.AlertsCreated = append(r.AlertsCreated, a)
	} else {
		r.AlertsMerged = append(r.AlertsMerged, a)
	}
}
