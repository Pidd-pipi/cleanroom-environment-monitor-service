package service

import (
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// ZoneService provides clean-zone and monitor-zone maintenance operations
// (creation, listing, detail views).
type ZoneService struct {
	store *store.Store
	audit *AuditService
}

// NewZoneService builds the zone service.
func NewZoneService(st *store.Store, audit *AuditService) *ZoneService {
	return &ZoneService{store: st, audit: audit}
}

// CreateCleanZone validates and creates a clean zone.
func (s *ZoneService) CreateCleanZone(z domain.CleanZone, operator, requestID string) (domain.CleanZone, error) {
	z.Touch()
	created, err := s.store.CleanZones().Create(z)
	if err != nil {
		return domain.CleanZone{}, err
	}
	_ = s.audit.Log(domain.AuditZoneCreate, operator, "clean_zone", z.ID,
		"clean zone created: "+z.Name, requestID)
	return created, nil
}

// CreateMonitorZone validates and creates a monitor zone.
func (s *ZoneService) CreateMonitorZone(m domain.MonitorZone, operator, requestID string) (domain.MonitorZone, error) {
	m.Touch()
	created, err := s.store.MonitorZones().Create(m)
	if err != nil {
		return domain.MonitorZone{}, err
	}
	_ = s.audit.Log(domain.AuditMonitorCreate, operator, "monitor_zone", m.ID,
		"monitor zone created: "+m.Name, requestID)
	return created, nil
}

// GetCleanZone returns a clean zone by id.
func (s *ZoneService) GetCleanZone(id string) (domain.CleanZone, error) {
	return s.store.CleanZones().Get(id)
}

// GetMonitorZone returns a monitor zone by id.
func (s *ZoneService) GetMonitorZone(id string) (domain.MonitorZone, error) {
	return s.store.MonitorZones().Get(id)
}

// ListCleanZones returns all clean zones.
func (s *ZoneService) ListCleanZones() ([]domain.CleanZone, error) {
	return s.store.CleanZones().List()
}

// ListMonitorZones returns all monitor zones.
func (s *ZoneService) ListMonitorZones() ([]domain.MonitorZone, error) {
	return s.store.MonitorZones().List()
}

// ListMonitorZonesByCleanZone returns the monitor zones of a clean zone.
func (s *ZoneService) ListMonitorZonesByCleanZone(cleanZoneID string) ([]domain.MonitorZone, error) {
	return s.store.MonitorZones().ListByCleanZone(cleanZoneID)
}

// SetMaintenance toggles the PM maintenance flag of a monitor zone and
// records the action in the audit trail (rule: PM affects data validity).
func (s *ZoneService) SetMaintenance(monitorZoneID string, inMaintenance bool, note, operator, requestID string) (domain.MonitorZone, error) {
	m, err := s.store.MonitorZones().Get(monitorZoneID)
	if err != nil {
		return domain.MonitorZone{}, err
	}
	m.SetMaintenance(inMaintenance, note)
	updated, err := s.store.MonitorZones().Update(m)
	if err != nil {
		return domain.MonitorZone{}, err
	}
	action := domain.AuditMaintenanceStart
	if !inMaintenance {
		action = domain.AuditMaintenanceEnd
	}
	_ = s.audit.Log(action, operator, "monitor_zone", monitorZoneID, note, requestID)
	return updated, nil
}

// SetCalibration updates the calibration due date of a monitor zone.
func (s *ZoneService) SetCalibration(monitorZoneID string, due time.Time, operator, requestID string) (domain.MonitorZone, error) {
	m, err := s.store.MonitorZones().Get(monitorZoneID)
	if err != nil {
		return domain.MonitorZone{}, err
	}
	m.Equipment.CalibrationDue = due.UTC()
	m.Touch()
	updated, err := s.store.MonitorZones().Update(m)
	if err != nil {
		return domain.MonitorZone{}, err
	}
	_ = s.audit.Log(domain.AuditMaintenanceEnd, operator, "monitor_zone", monitorZoneID,
		"calibration due updated to "+due.Format(time.RFC3339), requestID)
	return updated, nil
}
