package httpapi

import (
	"net/http"

	"example.com/cleanroom-environment-monitor-service/service"
)

// OverviewHandler serves the dashboard aggregate.
type OverviewHandler struct {
	svc *service.Services
}

// NewOverviewHandler builds the overview handler.
func NewOverviewHandler(svc *service.Services) *OverviewHandler {
	return &OverviewHandler{svc: svc}
}

// Get GET /api/overview
func (h *OverviewHandler) Get(w http.ResponseWriter, r *http.Request) {
	overview, err := h.svc.Overview.Build()
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, overview)
}
