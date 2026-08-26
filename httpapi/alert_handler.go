package httpapi

import (
	"net/http"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/service"
	"example.com/cleanroom-environment-monitor-service/store"
)

// AlertHandler serves the alert console endpoints.
type AlertHandler struct {
	svc *service.Services
}

// NewAlertHandler builds the alert handler.
func NewAlertHandler(svc *service.Services) *AlertHandler {
	return &AlertHandler{svc: svc}
}

// List GET /api/alerts?status=&type=&monitor_zone_id=&clean_zone_id=&limit=&offset=
func (h *AlertHandler) List(w http.ResponseWriter, r *http.Request) {
	filter := store.AlertFilter{}
	if s := r.URL.Query().Get("status"); s != "" {
		status, _ := domain.ParseAlertStatus(s)
		filter.Status = status
	}
	if t := r.URL.Query().Get("type"); t != "" {
		at, err := domain.ParseAlertType(t)
		if err != nil {
			Fail(w, r, err)
			return
		}
		filter.Type = at
	}
	if m := r.URL.Query().Get("monitor_zone_id"); m != "" {
		filter.MonitorZoneID = m
	}
	if z := r.URL.Query().Get("clean_zone_id"); z != "" {
		filter.CleanZoneID = z
	}
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	alerts, err := h.svc.Alerts.List(filter)
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(alerts)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, alerts[start:end], total, limit, offset)
}

// Get GET /api/alerts/{id}
func (h *AlertHandler) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	alert, err := h.svc.Store.Alerts().Get(id)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, alert)
}

// Ack POST /api/alerts/{id}/ack
func (h *AlertHandler) Ack(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var in struct {
		Operator    string `json:"operator"`
		Disposition string `json:"disposition"`
	}
	if err := decodeJSON(r, &in); err != nil {
		Fail(w, r, err)
		return
	}
	if in.Operator == "" {
		in.Operator = operatorFrom(r)
	}
	updated, err := h.svc.Alerts.Ack(id, in.Operator, in.Disposition, middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, updated)
}

// Escalate POST /api/alerts/{id}/escalate
func (h *AlertHandler) Escalate(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	updated, err := h.svc.Alerts.Escalate(id, operatorFrom(r), middlewareRequestID(r))
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, updated)
}

// Audit GET /api/audit?limit=&offset=&action=
func (h *AlertHandler) Audit(w http.ResponseWriter, r *http.Request) {
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	var action domain.AuditAction
	if a := r.URL.Query().Get("action"); a != "" {
		action = domain.AuditAction(a)
	}
	entries, err := h.svc.Audit.List(0, action)
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(entries)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, entries[start:end], total, limit, offset)
}
