package httpapi

import (
	"net/http"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/service"
)

// InterlockHandler serves interlock issuance, restore and query endpoints.
type InterlockHandler struct {
	svc *service.Services
}

// NewInterlockHandler builds the interlock handler.
func NewInterlockHandler(svc *service.Services) *InterlockHandler {
	return &InterlockHandler{svc: svc}
}

// Issue POST /api/zones/{id}/interlock
// Manually triggers an area-wide interlock. The body may carry the
// trigger monitor zone; when absent the first monitor zone of the clean
// zone is used as the trigger reference.
func (h *InterlockHandler) Issue(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("id")
	var in struct {
		TriggerMonitorZoneID string  `json:"trigger_monitor_zone_id"`
		Reason               string  `json:"reason"`
		Ratio                float64 `json:"ratio"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &in); err != nil {
			Fail(w, r, err)
			return
		}
	}
	if err := validateFinite("ratio", in.Ratio); err != nil {
		Fail(w, r, err)
		return
	}
	trigger := in.TriggerMonitorZoneID
	if trigger == "" {
		monitors, err := h.svc.Zones.ListMonitorZonesByCleanZone(zoneID)
		if err != nil {
			Fail(w, r, err)
			return
		}
		if len(monitors) > 0 {
			trigger = monitors[0].ID
		}
	}
	reason := in.Reason
	if reason == "" {
		reason = "manual_interlock"
	}
	ratio := in.Ratio
	if ratio < 1.0 {
		ratio = 1.5
	}
	logEntry, affected, err := h.svc.Interlock.IssueForArea(zoneID, trigger, reason, middlewareRequestID(r), ratio)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, map[string]interface{}{
		"interlock":      logEntry,
		"affected_zones": affected,
	})
}

// Restore POST /api/zones/{id}/restore
func (h *InterlockHandler) Restore(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("id")
	var in struct {
		Operator string `json:"operator"`
		Note     string `json:"note"`
	}
	if r.ContentLength > 0 {
		if err := decodeJSON(r, &in); err != nil {
			Fail(w, r, err)
			return
		}
	}
	if in.Operator == "" {
		in.Operator = operatorFrom(r)
	}
	if in.Note == "" {
		Fail(w, r, domain.InvalidInput("note is required to confirm restore"))
		return
	}
	restored, err := h.svc.Interlock.Restore(zoneID, in.Operator, in.Note, middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, map[string]interface{}{"restored_zones": restored})
}

// List GET /api/interlocks?limit=&offset=
func (h *InterlockHandler) List(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	logs, err := h.svc.Store.Interlocks().List()
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(logs)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, logs[start:end], total, limit, offset)
}

// ZoneInterlocks GET /api/zones/{id}/interlocks?limit=&offset=
func (h *InterlockHandler) ZoneInterlocks(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("id")
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	logs, err := h.svc.Store.Interlocks().ListByCleanZone(zoneID)
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(logs)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, logs[start:end], total, limit, offset)
}
