package service

import (
	"sort"
	"time"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/store"
)

// OverviewZone is one clean zone in the overview aggregate.
type OverviewZone struct {
	CleanZone      domain.CleanZone      `json:"clean_zone"`
	MonitorZones   []OverviewMonitorZone `json:"monitor_zones"`
	ActiveAlerts   int                   `json:"active_alerts"`
	Interlocked    bool                  `json:"interlocked"`
	PhysicalAreaID string                `json:"physical_area_id"`
}

// OverviewMonitorZone is one monitor zone with its latest sample.
type OverviewMonitorZone struct {
	MonitorZone  domain.MonitorZone `json:"monitor_zone"`
	LatestSample *domain.EnvSample  `json:"latest_sample,omitempty"`
	InvalidRatio float64            `json:"invalid_ratio"`
}

// Overview is the aggregate payload of GET /api/overview.
type Overview struct {
	Zones            []OverviewZone `json:"zones"`
	TotalZones       int            `json:"total_zones"`
	TotalMonitors    int            `json:"total_monitors"`
	ActiveAlerts     int            `json:"active_alerts"`
	OpenInterlocks   int            `json:"open_interlocks"`
	InterlockedZones int            `json:"interlocked_zones"`
	GeneratedAt      string         `json:"generated_at"`
}

// OverviewService aggregates the current state of every clean zone for the
// dashboard.
type OverviewService struct {
	cfg   *config.Config
	store *store.Store
}

// NewOverviewService builds the overview service.
func NewOverviewService(cfg *config.Config, st *store.Store) *OverviewService {
	return &OverviewService{cfg: cfg, store: st}
}

// Build computes the overview snapshot.
func (s *OverviewService) Build() (Overview, error) {
	zones, err := s.store.CleanZones().List()
	if err != nil {
		return Overview{}, err
	}
	monitors, err := s.store.MonitorZones().List()
	if err != nil {
		return Overview{}, err
	}
	activeByZone, err := s.store.Alerts().CountByCleanZoneActive()
	if err != nil {
		return Overview{}, err
	}
	openInterlocks, err := s.store.Interlocks().OpenCount()
	if err != nil {
		return Overview{}, err
	}

	monitorsByZone := map[string][]domain.MonitorZone{}
	for _, m := range monitors {
		monitorsByZone[m.CleanZoneID] = append(monitorsByZone[m.CleanZoneID], m)
	}

	out := Overview{GeneratedAt: nowString()}
	interlockedZones := 0
	for _, z := range zones {
		ovz := OverviewZone{
			CleanZone:      z,
			PhysicalAreaID: z.PhysicalArea,
			ActiveAlerts:   activeByZone[z.ID],
			Interlocked:    z.Status == domain.ZoneStatusInterlocked,
		}
		if ovz.Interlocked {
			interlockedZones++
		}
		for _, m := range monitorsByZone[z.ID] {
			omz := OverviewMonitorZone{MonitorZone: m}
			window, _ := s.store.Samples().RecentByMonitorZone(m.ID, s.cfg.SampleWindow())
			if len(window) == 0 {
				omz.InvalidRatio = 0
				ovz.MonitorZones = append(ovz.MonitorZones, omz)
				continue
			}
			sample := window[len(window)-1]
			omz.LatestSample = &sample
			ratio, _ := domain.EvaluateInvalidRatio(window, s.cfg.SampleWindow(), s.cfg.InvalidRatioThreshold)
			omz.InvalidRatio = ratio
			ovz.MonitorZones = append(ovz.MonitorZones, omz)
		}
		sort.SliceStable(ovz.MonitorZones, func(i, j int) bool {
			return ovz.MonitorZones[i].MonitorZone.ID < ovz.MonitorZones[j].MonitorZone.ID
		})
		out.Zones = append(out.Zones, ovz)
	}
	sort.SliceStable(out.Zones, func(i, j int) bool {
		return out.Zones[i].CleanZone.ID < out.Zones[j].CleanZone.ID
	})

	out.TotalZones = len(zones)
	out.TotalMonitors = len(monitors)
	out.ActiveAlerts = sumMap(activeByZone)
	out.OpenInterlocks = openInterlocks
	out.InterlockedZones = interlockedZones
	return out, nil
}

func sumMap(m map[string]int) int {
	total := 0
	for _, v := range m {
		total += v
	}
	return total
}

// nowString returns the current UTC time in RFC3339.
func nowString() string {
	return time.Now().UTC().Format(time.RFC3339)
}
