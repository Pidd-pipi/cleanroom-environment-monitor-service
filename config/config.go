// Package config holds all tunable knobs for the cleanroom monitor service:
// ISO threshold tables, process-specific threshold multipliers, timing rules,
// data-validity thresholds and environment overrides.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

// DefaultPort is used when the PORT environment variable is not set.
const DefaultPort = "8080"

// Config aggregates every tunable constant of the service. Each field is
// exported so tests and callers can construct custom configs.
type Config struct {
	// Port is the TCP port the HTTP server listens on (PORT env overrides).
	Port string

	// DataFile is the JSON persistence file. Empty disables persistence.
	DataFile string

	// LogLevel controls the process-wide structured logger
	// (debug/info/warn/error, LOG_LEVEL env overrides).
	LogLevel string

	// ShutdownTimeout is how long the graceful shutdown may take at most.
	ShutdownTimeout time.Duration

	// ReadHeaderTimeout / ReadTimeout / WriteTimeout / IdleTimeout are the
	// HTTP server timeouts (all must be positive).
	ReadHeaderTimeout time.Duration
	ReadTimeout       time.Duration
	WriteTimeout      time.Duration
	IdleTimeout       time.Duration

	// AutoInterlockAfter is how long a zone may stay above its particle
	// limit before the sweeper automatically issues an interlock.
	AutoInterlockAfter time.Duration

	// AlertEscalateAfter is how long an unacknowledged alert may live
	// before the sweeper escalates it.
	AlertEscalateAfter time.Duration

	// AlertDedupWindow is the merge window for repeated alerts of the
	// same type on the same monitor zone.
	AlertDedupWindow time.Duration

	// InvalidRatioThreshold is the share of invalid samples (within the
	// validity window) above which a data_quality alert is raised.
	InvalidRatioThreshold float64

	// InvalidRatioWindow is the number of recent samples used to compute
	// the invalid-data ratio.
	InvalidRatioWindow int

	// OverLimitRatio is the concentration-to-limit ratio that marks a
	// zone as over_limit (1.5 by spec: 1.5x of the limit).
	OverLimitRatio float64

	// InterlockSweepInterval is how often the zone over-limit sweeper runs.
	InterlockSweepInterval time.Duration

	// EscalateSweepInterval is how often the alert escalation sweeper runs.
	EscalateSweepInterval time.Duration

	// MaxSamplesPerZone caps the retained sample history per monitor zone.
	// A value of 0 means unbounded.
	MaxSamplesPerZone int
}

// Default returns a Config populated with the domain defaults from the
// prompt: 10 minutes auto-interlock, 1 hour escalation, 20 minutes
// dedup window, 30% invalid-data ratio and a 1.5x over-limit ratio.
func Default() *Config {
	return &Config{
		Port:                   DefaultPort,
		DataFile:               "data/cleanroom_data.json",
		LogLevel:               "info",
		ShutdownTimeout:        10 * time.Second,
		ReadHeaderTimeout:      10 * time.Second,
		ReadTimeout:            30 * time.Second,
		WriteTimeout:           60 * time.Second,
		IdleTimeout:            120 * time.Second,
		AutoInterlockAfter:     10 * time.Minute,
		AlertEscalateAfter:     1 * time.Hour,
		AlertDedupWindow:       20 * time.Minute,
		InvalidRatioThreshold:  0.30,
		InvalidRatioWindow:     50,
		OverLimitRatio:         1.5,
		InterlockSweepInterval: time.Minute,
		EscalateSweepInterval:  5 * time.Minute,
		MaxSamplesPerZone:      2000,
	}
}

// FromEnv builds a Config from environment variables, falling back to
// Default for anything unset. Invalid duration/number values are ignored and
// fall back to their defaults; use FromEnvStrict for fail-fast startup.
func FromEnv() *Config {
	cfg, _ := fromEnv(false)
	return cfg
}

// FromEnvStrict behaves like FromEnv but returns an error for the first
// invalid environment value so operators can reject a misconfigured process
// at startup instead of silently running with defaults.
func FromEnvStrict() (*Config, error) {
	return fromEnv(true)
}

func fromEnv(strict bool) (*Config, error) {
	cfg := Default()
	var err error
	assignString := func(target *string, key string) {
		if v := os.Getenv(key); v != "" {
			*target = v
		}
	}
	assignDuration := func(target *time.Duration, key string) {
		v := os.Getenv(key)
		if v == "" {
			return
		}
		d, perr := time.ParseDuration(v)
		if perr != nil {
			if strict && err == nil {
				err = fmt.Errorf("config: invalid %s=%q: %w", key, v, perr)
			}
			return
		}
		*target = d
	}
	assignFloat := func(target *float64, key string) {
		v := os.Getenv(key)
		if v == "" {
			return
		}
		f, perr := strconv.ParseFloat(v, 64)
		if perr != nil {
			if strict && err == nil {
				err = fmt.Errorf("config: invalid %s=%q: %w", key, v, perr)
			}
			return
		}
		*target = f
	}
	assignInt := func(target *int, key string) {
		v := os.Getenv(key)
		if v == "" {
			return
		}
		n, perr := strconv.Atoi(v)
		if perr != nil {
			if strict && err == nil {
				err = fmt.Errorf("config: invalid %s=%q: %w", key, v, perr)
			}
			return
		}
		*target = n
	}

	assignString(&cfg.Port, "PORT")
	assignString(&cfg.LogLevel, "LOG_LEVEL")
	if v, ok := os.LookupEnv("DATA_FILE"); ok {
		cfg.DataFile = v
	}
	assignDuration(&cfg.ShutdownTimeout, "SHUTDOWN_TIMEOUT")
	assignDuration(&cfg.ReadHeaderTimeout, "READ_HEADER_TIMEOUT")
	assignDuration(&cfg.ReadTimeout, "READ_TIMEOUT")
	assignDuration(&cfg.WriteTimeout, "WRITE_TIMEOUT")
	assignDuration(&cfg.IdleTimeout, "IDLE_TIMEOUT")
	assignDuration(&cfg.AutoInterlockAfter, "AUTO_INTERLOCK_AFTER")
	assignDuration(&cfg.AlertEscalateAfter, "ALERT_ESCALATE_AFTER")
	assignDuration(&cfg.AlertDedupWindow, "ALERT_DEDUP_WINDOW")
	assignDuration(&cfg.InterlockSweepInterval, "INTERLOCK_SWEEP_INTERVAL")
	assignDuration(&cfg.EscalateSweepInterval, "ESCALATE_SWEEP_INTERVAL")
	assignFloat(&cfg.InvalidRatioThreshold, "INVALID_RATIO_THRESHOLD")
	assignFloat(&cfg.OverLimitRatio, "OVER_LIMIT_RATIO")
	assignInt(&cfg.InvalidRatioWindow, "INVALID_RATIO_WINDOW")
	assignInt(&cfg.MaxSamplesPerZone, "MAX_SAMPLES_PER_ZONE")

	return cfg, err
}

// Validate checks// Validate checks that the configuration is internally consistent and
// returns a descriptive error otherwise.
func (c *Config) Validate() error {
	if c == nil {
		return fmt.Errorf("config: nil config")
	}
	if strings.TrimSpace(c.Port) == "" {
		return fmt.Errorf("config: Port must not be empty")
	}
	switch strings.ToLower(strings.TrimSpace(c.LogLevel)) {
	case "", "debug", "info", "warn", "warning", "error":
	default:
		return fmt.Errorf("config: invalid LogLevel %q (want debug/info/warn/error)", c.LogLevel)
	}
	for name, d := range map[string]time.Duration{
		"ShutdownTimeout":        c.ShutdownTimeout,
		"ReadHeaderTimeout":      c.ReadHeaderTimeout,
		"ReadTimeout":            c.ReadTimeout,
		"WriteTimeout":           c.WriteTimeout,
		"IdleTimeout":            c.IdleTimeout,
		"AutoInterlockAfter":     c.AutoInterlockAfter,
		"AlertEscalateAfter":     c.AlertEscalateAfter,
		"AlertDedupWindow":       c.AlertDedupWindow,
		"InterlockSweepInterval": c.InterlockSweepInterval,
		"EscalateSweepInterval":  c.EscalateSweepInterval,
	} {
		if d <= 0 {
			return fmt.Errorf("config: %s must be positive", name)
		}
	}
	if c.InvalidRatioThreshold <= 0 || c.InvalidRatioThreshold > 1 {
		return fmt.Errorf("config: InvalidRatioThreshold must be in (0,1]")
	}
	if c.InvalidRatioWindow < 1 {
		return fmt.Errorf("config: InvalidRatioWindow must be >= 1")
	}
	if c.OverLimitRatio < 1 {
		return fmt.Errorf("config: OverLimitRatio must be >= 1")
	}
	if c.MaxSamplesPerZone < 0 {
		return fmt.Errorf("config: MaxSamplesPerZone must be >= 0")
	}
	return nil
}

// SampleWindow is a helper returning the time window (in samples) that the
// invalid-ratio computation uses, so callers do not dereference nil configs.
func (c *Config) SampleWindow() int {
	if c == nil || c.InvalidRatioWindow < 1 {
		return 50
	}
	return c.InvalidRatioWindow
}

// ISO14644Limits returns the standard ISO 14644-1 concentration limits
// (particles per cubic metre) for the supported ISO classes.
func ISO14644Limits() map[domain.IsoClass]domain.IsoLimit {
	return map[domain.IsoClass]domain.IsoLimit{
		domain.Iso5: {Count0303: 100000, Count0505: 35000},
		domain.Iso6: {Count0303: 1000000, Count0505: 350000},
		domain.Iso7: {Count0303: 10000000, Count0505: 3500000},
		domain.Iso8: {Count0303: 100000000, Count0505: 35000000},
	}
}
