package httpapi

import (
	"net/http"

	"example.com/cleanroom-environment-monitor-service/domain"
	"example.com/cleanroom-environment-monitor-service/service"
)

// SampleHandler serves sample ingestion and trend queries.
type SampleHandler struct {
	svc *service.Services
}

// NewSampleHandler builds the sample handler.
func NewSampleHandler(svc *service.Services) *SampleHandler {
	return &SampleHandler{svc: svc}
}

// sampleRequest is the JSON payload of POST /api/monitors/{id}/samples.
type sampleRequest struct {
	Count0303    float64 `json:"count_0_3um"`
	Count0505    float64 `json:"count_0_5um"`
	Temperature  float64 `json:"temperature"`
	Humidity     float64 `json:"humidity"`
	PressureDiff float64 `json:"pressure_diff"`
	Timestamp    string  `json:"timestamp"`
}

// PostSample POST /api/monitors/{id}/samples
func (h *SampleHandler) PostSample(w http.ResponseWriter, r *http.Request) {
	monitorID := r.PathValue("id")
	var in sampleRequest
	if err := decodeJSON(r, &in); err != nil {
		Fail(w, r, err)
		return
	}
	if err := validateSamplePayload(in); err != nil {
		Fail(w, r, err)
		return
	}
	ts, err := parseTime(in.Timestamp)
	if err != nil {
		Fail(w, r, err)
		return
	}
	req := service.IngestRequest{
		MonitorZoneID: monitorID,
		Count0303:     in.Count0303,
		Count0505:     in.Count0505,
		Temperature:   in.Temperature,
		Humidity:      in.Humidity,
		PressureDiff:  in.PressureDiff,
		Timestamp:     ts,
		Operator:      operatorFrom(r),
		RequestID:     middlewareRequestID(r),
	}
	result, err := h.svc.Ingest.Process(req)
	if err != nil {
		Fail(w, r, err)
		return
	}
	OK(w, r, result)
}

// validateSamplePayload performs complete payload validation so malformed
// sensor readings produce a 400 before reaching the domain, and can never
// panic the handler.
func validateSamplePayload(in sampleRequest) error {
	for name, v := range map[string]float64{
		"count_0_3um":   in.Count0303,
		"count_0_5um":   in.Count0505,
		"temperature":   in.Temperature,
		"humidity":      in.Humidity,
		"pressure_diff": in.PressureDiff,
	} {
		if err := validateFinite(name, v); err != nil {
			return err
		}
	}
	if in.Count0303 < 0 || in.Count0505 < 0 {
		return domain.InvalidInput("particle counts must be non-negative")
	}
	if in.Temperature < -100 || in.Temperature > 200 {
		return domain.InvalidInput("temperature is outside the plausible range -100..200 °C")
	}
	if in.Humidity < 0 || in.Humidity > 100 {
		return domain.InvalidInput("humidity must be within 0..100 %")
	}
	if in.PressureDiff < -100000 || in.PressureDiff > 100000 {
		return domain.InvalidInput("pressure_diff is outside the plausible range -100000..100000 Pa")
	}
	return nil
}

// ZoneSamples GET /api/zones/{id}/samples?limit=&offset=
func (h *SampleHandler) ZoneSamples(w http.ResponseWriter, r *http.Request) {
	zoneID := r.PathValue("id")
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	samples, err := h.svc.Store.Samples().ListByCleanZone(zoneID, 0)
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(samples)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, samples[start:end], total, limit, offset)
}

// MonitorSamples GET /api/monitors/{id}/samples?limit=&offset=
func (h *SampleHandler) MonitorSamples(w http.ResponseWriter, r *http.Request) {
	monitorID := r.PathValue("id")
	limit, offset, err := parsePagination(r)
	if err != nil {
		Fail(w, r, err)
		return
	}
	if _, err := h.svc.Zones.GetMonitorZone(monitorID); err != nil {
		Fail(w, r, err)
		return
	}
	samples, err := h.svc.Store.Samples().ListByMonitorZone(monitorID, 0)
	if err != nil {
		Fail(w, r, err)
		return
	}
	total := len(samples)
	start, end := paginate(total, limit, offset)
	OKPaged(w, r, samples[start:end], total, limit, offset)
}
