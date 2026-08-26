package config

import (
	"os"
	"testing"
	"time"

	"example.com/cleanroom-environment-monitor-service/domain"
)

func TestDefaultConfig(t *testing.T) {
	cfg := Default()
	if cfg.Port != "8080" {
		t.Fatalf("expected default port 8080, got %s", cfg.Port)
	}
	if cfg.AutoInterlockAfter != 10*time.Minute {
		t.Fatalf("expected auto interlock 10m, got %v", cfg.AutoInterlockAfter)
	}
	if cfg.AlertEscalateAfter != time.Hour {
		t.Fatalf("expected escalation 1h, got %v", cfg.AlertEscalateAfter)
	}
	if cfg.AlertDedupWindow != 20*time.Minute {
		t.Fatalf("expected dedup 20m, got %v", cfg.AlertDedupWindow)
	}
	if cfg.InvalidRatioThreshold != 0.30 {
		t.Fatalf("expected invalid ratio 0.3, got %v", cfg.InvalidRatioThreshold)
	}
	if cfg.OverLimitRatio != 1.5 {
		t.Fatalf("expected over-limit ratio 1.5, got %v", cfg.OverLimitRatio)
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("default config must validate: %v", err)
	}
}

func TestConfigValidateRejectsBadValues(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Config)
	}{
		{"empty port", func(c *Config) { c.Port = "" }},
		{"zero interlock", func(c *Config) { c.AutoInterlockAfter = 0 }},
		{"zero escalate", func(c *Config) { c.AlertEscalateAfter = 0 }},
		{"zero dedup", func(c *Config) { c.AlertDedupWindow = 0 }},
		{"ratio zero", func(c *Config) { c.InvalidRatioThreshold = 0 }},
		{"ratio over 1", func(c *Config) { c.InvalidRatioThreshold = 1.2 }},
		{"overlimit below 1", func(c *Config) { c.OverLimitRatio = 0.9 }},
		{"empty window", func(c *Config) { c.InvalidRatioWindow = 0 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := Default()
			tc.mutate(cfg)
			if err := cfg.Validate(); err == nil {
				t.Fatalf("expected validation error for %s", tc.name)
			}
		})
	}
}

func TestFromEnvOverrides(t *testing.T) {
	os.Setenv("PORT", "9999")
	os.Setenv("DATA_FILE", "")
	os.Setenv("AUTO_INTERLOCK_AFTER", "5m")
	os.Setenv("ALERT_ESCALATE_AFTER", "2h")
	os.Setenv("ALERT_DEDUP_WINDOW", "10m")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("DATA_FILE")
		os.Unsetenv("AUTO_INTERLOCK_AFTER")
		os.Unsetenv("ALERT_ESCALATE_AFTER")
		os.Unsetenv("ALERT_DEDUP_WINDOW")
	}()
	cfg := FromEnv()
	if cfg.Port != "9999" {
		t.Fatalf("expected PORT override 9999, got %s", cfg.Port)
	}
	if cfg.DataFile != "" {
		t.Fatalf("expected DATA_FILE empty override, got %q", cfg.DataFile)
	}
	if cfg.AutoInterlockAfter != 5*time.Minute {
		t.Fatalf("expected 5m auto interlock, got %v", cfg.AutoInterlockAfter)
	}
	if cfg.AlertEscalateAfter != 2*time.Hour {
		t.Fatalf("expected 2h escalation, got %v", cfg.AlertEscalateAfter)
	}
	if cfg.AlertDedupWindow != 10*time.Minute {
		t.Fatalf("expected 10m dedup, got %v", cfg.AlertDedupWindow)
	}
}

func TestISO14644LimitsComplete(t *testing.T) {
	table := ISO14644Limits()
	for _, cls := range domain.AllIsoClasses() {
		lim, ok := table[cls]
		if !ok {
			t.Fatalf("missing ISO limit for %s", cls)
		}
		if lim.Count0303 <= 0 || lim.Count0505 <= 0 {
			t.Fatalf("non-positive limit for %s", cls)
		}
	}
	// Cleaner classes must be stricter than dirtier ones.
	if table[domain.Iso5].Count0303 >= table[domain.Iso6].Count0303 {
		t.Fatal("ISO5 must be stricter than ISO6")
	}
}

func TestProcessThresholdsComplete(t *testing.T) {
	if err := ValidateProcessThresholds(ProcessThresholds()); err != nil {
		t.Fatalf("process thresholds invalid: %v", err)
	}
	if len(ProcessThresholds()) != len(domain.AllProcessTypes()) {
		t.Fatal("process threshold count mismatch")
	}
}
