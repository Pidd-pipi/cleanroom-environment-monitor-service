package httpapi

import (
	"net/http"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/service"
)

// ZoneHandler serves clean-zone and monitor-zone endpoints.
type ZoneHandler struct {
	svc *service.Services
}

// NewZoneHandler builds the zone handler.
func NewZoneHandler(svc *service.Services) *ZoneHandler {
	return &ZoneHandler{svc: svc}
}

// ListCleanZones GET /api/zones?limit=&offset=
func (h *ZoneHandler) ListCleanZones(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	zones, err := h.svc.Zones.ListCleanZones()
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(zones)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, zones[start:end], total, limit, offset)
}

// GetCleanZone GET /api/zones/{id}
func (h *ZoneHandler) GetCleanZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	zone, err := h.svc.Zones.GetCleanZone(id)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, zone)
}

// CreateCleanZone POST /api/zones
func (h *ZoneHandler) CreateCleanZone(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID           string             `json:"id"`
		Name         string             `json:"name"`
		PhysicalArea string             `json:"physical_area"`
		IsoClass     domain.IsoClass    `json:"iso_class"`
		Process      domain.ProcessType `json:"process"`
	}
	if err := decodeJSON(r, &in); err != nil {
		Fail(w, r, err)
		return
	}
	zone := domain.NewCleanZone(in.ID, in.Name, in.PhysicalArea, in.IsoClass, in.Process)
	if in.ID == "" {
		zone.ID = h.svc.Store.NewID("zone")
	}
	created, err := h.svc.Zones.CreateCleanZone(zone, operatorFrom(r), middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OKStatus(w, r, http.StatusCreated, created)
}

// ListMonitorZones GET /api/monitors?limit=&offset=
func (h *ZoneHandler) ListMonitorZones(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	monitors, err := h.svc.Zones.ListMonitorZones()
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(monitors)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, monitors[start:end], total, limit, offset)
}

// GetMonitorZone GET /api/monitors/{id}
func (h *ZoneHandler) GetMonitorZone(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	m, err := h.svc.Zones.GetMonitorZone(id)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, m)
}

// CreateMonitorZone POST /api/monitors
func (h *ZoneHandler) CreateMonitorZone(w http.ResponseWriter, r *http.Request) {
	var in struct {
		ID                string                `json:"id"`
		CleanZoneID       string                `json:"clean_zone_id"`
		Name              string                `json:"name"`
		ParticleCounterID string                `json:"particle_counter_id"`
		Thresholds        domain.ZoneThresholds `json:"thresholds"`
	}
	if err := decodeJSON(r, &in); err != nil {
		Fail(w, r, err)
		return
	}
	if err := validateThresholds(in.Thresholds); err != nil {
		Fail(w, r, err)
		return
	}
	m := domain.NewMonitorZone(in.ID, in.CleanZoneID, in.Name, in.ParticleCounterID)
	if in.ID == "" {
		m.ID = h.svc.Store.NewID("monitor")
	}
	m.Thresholds = in.Thresholds
	created, err := h.svc.Zones.CreateMonitorZone(m, operatorFrom(r), middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OKStatus(w, r, http.StatusCreated, created)
}

// SetMaintenance POST /api/monitors/{id}/maintenance
func (h *ZoneHandler) SetMaintenance(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		InMaintenance bool   `json:"in_maintenance"`
		Note          string `json:"note"`
	}
	if err := decodeJSON(r, &in); err != nil {
		Fail(w, r, err)
		return
	}
	updated, err := h.svc.Zones.SetMaintenance(id, in.InMaintenance, in.Note, operatorFrom(r), middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, updated)
}

// SetCalibration POST /api/monitors/{id}/calibration
func (h *ZoneHandler) SetCalibration(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		CalibrationDue string `json:"calibration_due"` // RFC3339
	}
	if err := decodeJSON(r, &in); err != nil {
		Fail(w, r, err)
		return
	}
	due, err := parseTime(in.CalibrationDue)
	if err != nil {
		Fail(w, r, err)
		return
	}
	updated, err := h.svc.Zones.SetCalibration(id, due, operatorFrom(r), middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, updated)
}

// validateThresholds rejects non-finite or nonsensical threshold overrides
// before they enter the domain, keeping invalid input as a 400.
func validateThresholds(t domain.ZoneThresholds) error {
	for name, v := range map[string]float64{
		"limit_0_3um":  t.Limit0303,
		"limit_0_5um":  t.Limit0505,
		"temp_min":     t.TempMin,
		"temp_max":     t.TempMax,
		"humidity_min": t.HumidityMin,
		"humidity_max": t.HumidityMax,
		"pressure_min": t.PressureMin,
		"pressure_max": t.PressureMax,
	} {
		if err := validateFinite(name, v); err != nil {
			return err
		}
	}
	if t.Limit0303 < 0 || t.Limit0505 < 0 {
		return domain.InvalidInput("particle limit overrides must be non-negative")
	}
	if t.TempMin != 0 && t.TempMax != 0 && t.TempMin >= t.TempMax {
		return domain.InvalidInput("temp_min must be lower than temp_max")
	}
	if t.HumidityMin != 0 && t.HumidityMax != 0 && t.HumidityMin >= t.HumidityMax {
		return domain.InvalidInput("humidity_min must be lower than humidity_max")
	}
	if t.PressureMin != 0 && t.PressureMax != 0 && t.PressureMin >= t.PressureMax {
		return domain.InvalidInput("pressure_min must be lower than pressure_max")
	}
	return nil
}

// operatorFrom returns the acting operator from a header (no auth yet).
func operatorFrom(r *http.Request) string {
	if op := r.Header.Get("X-Operator"); op != "" {
		return op
	}
	return "web_user"
}

// middlewareRequestID reads the trace id from the request context.
func middlewareRequestID(r *http.Request) string {
	return requestIDFrom(r)
}
