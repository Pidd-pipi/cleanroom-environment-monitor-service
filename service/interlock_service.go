package service

import (
	"fmt"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// InterlockService orchestrates interlock issuance and restore with the
// whole-physical-area consistency rule.
type InterlockService struct {
	cfg   *config.Config
	store *store.Store
	audit *AuditService
}

// NewInterlockService builds the interlock service.
func NewInterlockService(cfg *config.Config, st *store.Store, audit *AuditService) *InterlockService {
	return &InterlockService{cfg: cfg, store: st, audit: audit}
}

// IssueForArea triggers an interlock for the physical area containing the
// given clean zone. Per rule 6 every clean zone in the same physical area
// enters the interlocked state, not just the trigger zone.
func (s *InterlockService) IssueForArea(cleanZoneID, triggerMonitorZoneID, reason, requestID string, peakRatio float64) (domain.InterlockLog, []domain.CleanZone, error) {
	zone, err := s.store.CleanZones().Get(cleanZoneID)
	if err != nil {
		return domain.InterlockLog{}, nil, err
	}
	areaZones, err := s.store.CleanZones().ListByPhysicalArea(zone.PhysicalArea)
	if err != nil {
		return domain.InterlockLog{}, nil, err
	}
	if len(areaZones) == 0 {
		return domain.InterlockLog{}, nil, domain.NotFound("clean zone", cleanZoneID)
	}

	level := domain.InterlockLevelForRatio(peakRatio)
	actions := domain.ActionsForLevel(level)
	now := time.Now().UTC()

	affected := make([]string, 0, len(areaZones))
	for i := range areaZones {
		z := &areaZones[i]
		if z.Status == domain.ZoneStatusInterlocked {
			affected = append(affected, z.ID)
			continue
		}
		if !domain.CanTransition(z.Status, domain.ZoneStatusInterlocked) {
			return domain.InterlockLog{}, nil, domain.Conflict(
				fmt.Sprintf("clean zone %s cannot interlock from state %s", z.ID, z.Status))
		}
		z.SetStatus(domain.ZoneStatusInterlocked)
		affected = append(affected, z.ID)
		if _, err := s.store.CleanZones().Update(*z); err != nil {
			return domain.InterlockLog{}, nil, err
		}
	}

	// Apply the FFU / fresh-air commands to every monitor zone in the area.
	if err := s.applyInterlockCommands(zone.PhysicalArea, actions, level); err != nil {
		return domain.InterlockLog{}, nil, err
	}

	// One interlock log per area; the log lists every affected zone.
	logEntry := domain.NewInterlockLog(
		s.store.NewID("interlock"),
		zone.ID,
		zone.PhysicalArea,
		triggerMonitorZoneID,
		affected,
		actions[0],
		level,
		reason,
		peakRatio,
	)
	logEntry.IssuedAt = now
	created, err := s.store.Interlocks().Create(logEntry)
	if err != nil {
		return domain.InterlockLog{}, nil, err
	}

	detail := fmt.Sprintf("interlock level=%d actions=%v peak_ratio=%.3f affected=%v",
		level, actions, peakRatio, affected)
	_ = s.audit.Log(domain.AuditInterlockIssue, "system", "clean_zone", zone.ID, detail, requestID)
	return created, areaZones, nil
}

// applyInterlockCommands raises FFU speed and fresh-air ratio on every
// monitor zone of a physical area.
func (s *InterlockService) applyInterlockCommands(physicalArea string, actions []domain.InterlockAction, level int) error {
	zones, err := s.store.CleanZones().ListByPhysicalArea(physicalArea)
	if err != nil {
		return err
	}
	zoneIDs := map[string]bool{}
	for _, z := range zones {
		zoneIDs[z.ID] = true
	}
	monitors, err := s.store.MonitorZones().List()
	if err != nil {
		return err
	}
	for _, m := range monitors {
		if !zoneIDs[m.CleanZoneID] {
			continue
		}
		for _, a := range actions {
			switch a {
			case domain.InterlockFFUSpeedUp:
				m.Equipment.FFULevel = minInt(100, m.Equipment.FFULevel+10*level)
			case domain.InterlockFreshAirIncrease:
				m.Equipment.FreshAirRatio = minInt(100, m.Equipment.FreshAirRatio+5*level)
			}
		}
		m.Touch()
		if _, err := s.store.MonitorZones().Update(m); err != nil {
			return err
		}
	}
	return nil
}

// Restore confirms recovery of a clean zone: per the state machine the zone
// moves to restored and every open interlock log of the physical area is
// closed. Only an interlocked zone may be restored; restoring a clean or
// over-limit zone is rejected so operators cannot "confirm" a recovery that
// never interlocked, and the open interlock logs of the whole physical area
// (not only the triggering zone's log) are closed.
func (s *InterlockService) Restore(cleanZoneID, operator, note, requestID string) ([]domain.CleanZone, error) {
	zone, err := s.store.CleanZones().Get(cleanZoneID)
	if err != nil {
		return nil, err
	}

	// A restore confirmation is only meaningful for a zone that is currently
	// interlocked. Over-limit, elevated, normal or restored zones have no
	// active interlock to confirm: silently succeeding here would let an
	// operator "restore" a clean zone and mask a real interlock elsewhere.
	if zone.Status != domain.ZoneStatusInterlocked {
		if zone.Status == domain.ZoneStatusRestored {
			// Idempotent re-confirm: nothing to do, no logs to close.
			return []domain.CleanZone{zone}, nil
		}
		return nil, domain.Conflict(
			fmt.Sprintf("clean zone %s is %s; only an interlocked zone can be restored", zone.ID, zone.Status))
	}

	areaZones, err := s.store.CleanZones().ListByPhysicalArea(zone.PhysicalArea)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	restored := make([]domain.CleanZone, 0, len(areaZones))
	for i := range areaZones {
		z := &areaZones[i]
		if z.Status == domain.ZoneStatusRestored {
			restored = append(restored, *z)
			continue
		}
		if !domain.CanTransition(z.Status, domain.RestoreStatus) {
			continue
		}
		z.SetStatus(domain.RestoreStatus)
		if _, err := s.store.CleanZones().Update(*z); err != nil {
			return nil, err
		}
		restored = append(restored, *z)
	}

	// Close every open interlock log of the physical area. The triggering
	// zone recorded in the log (CleanZoneID) may be any zone of the area, so
	// restoring from a sibling zone must still close the trigger's log.
	logs, err := s.store.Interlocks().List()
	if err != nil {
		return nil, err
	}
	for _, l := range logs {
		if !l.IsOpen() || l.PhysicalArea != zone.PhysicalArea {
			continue
		}
		l.Close(operator, note, now)
		if _, err := s.store.Interlocks().Update(l); err != nil {
			return nil, err
		}
	}

	detail := fmt.Sprintf("restore confirmed for physical area %s zones=%v", zone.PhysicalArea, len(restored))
	_ = s.audit.Log(domain.AuditZoneRestore, operator, "clean_zone", zone.ID, detail, requestID)
	return restored, nil
}

// AutoInterlockCandidates returns clean zones that have stayed elevated or
// over_limit longer than the auto-interlock timeout (rule: over-limit 10
// minutes without recovery enters interlocked ventilation automatically).
func (s *InterlockService) AutoInterlockCandidates(now time.Time) ([]domain.CleanZone, error) {
	zones, err := s.store.CleanZones().List()
	if err != nil {
		return nil, err
	}
	out := make([]domain.CleanZone, 0)
	for _, z := range zones {
		if z.Status != domain.ZoneStatusElevated && z.Status != domain.ZoneStatusOverLimit {
			continue
		}
		if now.Sub(z.StatusSince) >= s.cfg.AutoInterlockAfter {
			out = append(out, z)
		}
	}
	return out, nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
