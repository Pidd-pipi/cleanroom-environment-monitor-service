// Package logging configures the process-wide structured logger used by all
// packages. It uses only the standard library log/slog so the
// service keeps its zero-dependency property.
package logging

import (
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ParseLevel converts a LOG_LEVEL-style string into a slog.Level. An empty
// string maps to slog.LevelInfo, the production default.
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "info":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error":
		return slog.LevelError, nil
	default:
		return 0, fmt.Errorf("unknown log level %q (want debug, info, warn or error)", s)
	}
}

// Configure installs a text-handler logger as slog.Default() and applies the
// requested level filter. Every package that uses slog.Info/slog.Warn/
// slog.Error therefore shares the same format and level control.
func Configure(level string) error {
	lvl, err := ParseLevel(level)
	if err != nil {
		return err
	}
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: lvl,
	})))
	return nil
}
