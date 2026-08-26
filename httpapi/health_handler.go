package httpapi

import (
	"net/http"
	"time"

	"example.com/cleanroom-environment-monitor-service/store"
)

// HealthHandler serves the health-check endpoint used by runtime smoke
// tests and load balancers.
type HealthHandler struct {
	store   *store.Store
	started time.Time
}

// NewHealthHandler builds the health handler.
func NewHealthHandler(st *store.Store) *HealthHandler {
	return &HealthHandler{store: st, started: time.Now()}
}

// Healthz responds 200 with service metadata when the service is ready.
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	auditCount, _ := h.store.Audit().Count()
	zoneCount, _ := h.store.CleanZones().Count()
	OK(w, r, map[string]interface{}{
		"status":       "ok",
		"uptime_secs":  int64(time.Since(h.started).Seconds()),
		"zones":        zoneCount,
		"audit_events": auditCount,
		"time":         time.Now().UTC().Format(time.RFC3339),
	})
}
