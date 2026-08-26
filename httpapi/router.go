package httpapi

import (
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"strings"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/middleware"
	"example.com/cleanroom-environment-monitor-service/service"
	"example.com/cleanroom-environment-monitor-service/store"
)

// Handler is the minimal contract every API handler implements.
type Handler interface {
	Register(mux *http.ServeMux)
}

// Router builds the full HTTP handler: middleware chain, REST API and the
// embedded SPA static server.
type Router struct {
	cfg      *config.Config
	store    *store.Store
	svc      *service.Services
	webFS    fs.FS
	staticFS fs.FS
}

// NewRouter constructs the router with the embedded web filesystem. It keeps
// the original single-return signature for backward compatibility and panics
// only when the embedded filesystem is misconfigured (which cannot happen
// for the production go:embed layout).
func NewRouter(cfg *config.Config, st *store.Store, svc *service.Services, webFS fs.FS) *Router {
	r, err := NewRouterE(cfg, st, svc, webFS)
	if err != nil {
		slog.Error("httpapi: embed web fs", "error", err)
		panic(err)
	}
	return r
}

// NewRouterE is the error-returning constructor used by main so startup
// failures can be logged with the structured logger and handled gracefully.
func NewRouterE(cfg *config.Config, st *store.Store, svc *service.Services, webFS fs.FS) (*Router, error) {
	static, err := fs.Sub(webFS, "web")
	if err != nil {
		return nil, fmt.Errorf("httpapi: embed web fs: %w", err)
	}
	return &Router{cfg: cfg, store: st, svc: svc, webFS: webFS, staticFS: static}, nil
}

// Handler assembles the middleware-wrapped mux.
func (r *Router) Handler() http.Handler {
	mux := http.NewServeMux()
	r.registerAPI(mux)
	r.registerWeb(mux)

	audit := middleware.NewAuditLogger(r.store, "/api/healthz", "/healthz", "/readyz", "/assets/", "/components/", "/hooks/", "/pages/")
	chain := middleware.RequestIDMiddleware(
		middleware.SecurityHeadersMiddleware(
			middleware.RequestLogMiddleware(
				middleware.PanicRecoveryMiddleware(audit.Wrap(mux)))))
	return chain
}

// registerAPI wires every REST endpoint. Routes use Go 1.22+ path
// parameters via http.ServeMux.
func (r *Router) registerAPI(mux *http.ServeMux) {
	health := NewHealthHandler(r.store)
	zones := NewZoneHandler(r.svc)
	samples := NewSampleHandler(r.svc)
	interlocks := NewInterlockHandler(r.svc)
	alerts := NewAlertHandler(r.svc)
	overview := NewOverviewHandler(r.svc)

	mux.HandleFunc("GET /healthz", health.Healthz)
	mux.HandleFunc("GET /readyz", health.Healthz)
	mux.HandleFunc("GET /api/healthz", health.Healthz)

	mux.HandleFunc("GET /api/overview", overview.Get)

	mux.HandleFunc("GET /api/zones", zones.ListCleanZones)
	mux.HandleFunc("POST /api/zones", zones.CreateCleanZone)
	mux.HandleFunc("GET /api/zones/{id}", zones.GetCleanZone)
	mux.HandleFunc("GET /api/zones/{id}/samples", samples.ZoneSamples)
	mux.HandleFunc("GET /api/zones/{id}/interlocks", interlocks.ZoneInterlocks)
	mux.HandleFunc("POST /api/zones/{id}/interlock", interlocks.Issue)
	mux.HandleFunc("POST /api/zones/{id}/restore", interlocks.Restore)

	mux.HandleFunc("GET /api/monitors", zones.ListMonitorZones)
	mux.HandleFunc("POST /api/monitors", zones.CreateMonitorZone)
	mux.HandleFunc("GET /api/monitors/{id}", zones.GetMonitorZone)
	mux.HandleFunc("GET /api/monitors/{id}/samples", samples.MonitorSamples)
	mux.HandleFunc("POST /api/monitors/{id}/samples", samples.PostSample)
	mux.HandleFunc("POST /api/monitors/{id}/maintenance", zones.SetMaintenance)
	mux.HandleFunc("POST /api/monitors/{id}/calibration", zones.SetCalibration)

	mux.HandleFunc("GET /api/alerts", alerts.List)
	mux.HandleFunc("GET /api/alerts/{id}", alerts.Get)
	mux.HandleFunc("POST /api/alerts/{id}/ack", alerts.Ack)
	mux.HandleFunc("POST /api/alerts/{id}/escalate", alerts.Escalate)

	mux.HandleFunc("GET /api/interlocks", interlocks.List)
	mux.HandleFunc("GET /api/audit", alerts.Audit)
}

// registerWeb serves the embedded SPA. Static assets are served directly;
// unknown non-API paths fall back to index.html for client-side routing.
func (r *Router) registerWeb(mux *http.ServeMux) {
	fileServer := http.FileServer(http.FS(r.staticFS))
	mux.HandleFunc("/", func(w http.ResponseWriter, req *http.Request) {
		path := req.URL.Path
		if strings.HasPrefix(path, "/api/") {
			FailPlain(w, req, http.StatusNotFound, "api route not found: "+path)
			return
		}
		// Serve an actual static file when it exists in the embedded FS.
		if path != "/" {
			if f, err := r.staticFS.Open(strings.TrimPrefix(path, "/")); err == nil {
				f.Close()
				fileServer.ServeHTTP(w, req)
				return
			}
		}
		// SPA fallback: always serve the app shell.
		data, err := fs.ReadFile(r.staticFS, "index.html")
		if err != nil {
			FailPlain(w, req, http.StatusInternalServerError, "app shell missing")
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}
