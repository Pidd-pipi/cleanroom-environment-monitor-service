// Command cleanroom-environment-monitor-service is a Go web service for
// semiconductor cleanroom environment monitoring: zone management, particle
// and climate data ingestion, ISO classification, area-wide interlocks and
// alert handling. It embeds a native HTML/CSS/JS frontend via go:embed.
package main

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"example.com/cleanroom-environment-monitor-service/config"
	"example.com/cleanroom-environment-monitor-service/httpapi"
	"example.com/cleanroom-environment-monitor-service/logging"
	"example.com/cleanroom-environment-monitor-service/service"
	"example.com/cleanroom-environment-monitor-service/store"
)

//go:embed all:web
var webFS embed.FS

func main() {
	cfg, err := config.FromEnvStrict()
	if err != nil {
		fmt.Fprintf(os.Stderr, "main: invalid environment: %v\n", err)
		os.Exit(1)
	}
	if err := logging.Configure(cfg.LogLevel); err != nil {
		fmt.Fprintf(os.Stderr, "main: configure logging: %v\n", err)
		os.Exit(1)
	}
	if err := cfg.Validate(); err != nil {
		slog.Error("main: invalid config", "error", err)
		os.Exit(1)
	}

	st := store.NewStore(cfg.DataFile)
	if err := st.Load(); err != nil {
		slog.Error("main: load store", "error", err)
		os.Exit(1)
	}
	if warn := st.LoadWarning(); warn != nil {
		slog.Warn("main: degraded store start", "warning", warn)
	}

	svc := service.New(cfg, st)
	boot := service.NewBootstrap(cfg, st, svc.Ingest)
	if err := boot.SeedIfEmpty(); err != nil {
		slog.Error("main: bootstrap", "error", err)
		os.Exit(1)
	}

	router, err := httpapi.NewRouterE(cfg, st, svc, webFS)
	if err != nil {
		slog.Error("main: build router", "error", err)
		os.Exit(1)
	}
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router.Handler(),
		ReadHeaderTimeout: cfg.ReadHeaderTimeout,
		ReadTimeout:       cfg.ReadTimeout,
		WriteTimeout:      cfg.WriteTimeout,
		IdleTimeout:       cfg.IdleTimeout,
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Background sweeper goroutines stop when the context is cancelled.
	svc.StartSweepers(ctx)

	errCh := make(chan error, 1)
	go func() {
		slog.Info("main: cleanroom monitor service listening", "port", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh:
		slog.Error("main: server error", "error", err)
		os.Exit(1)
	case <-ctx.Done():
		slog.Info("main: shutting down")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			slog.Error("main: shutdown", "error", err)
		}
		if cfg.DataFile != "" {
			if err := st.Save(); err != nil {
				slog.Error("main: final save", "error", err)
			} else {
				slog.Info("main: persisted state", "file", cfg.DataFile)
			}
		}
		slog.Info("main: bye")
	}
}
