package config

import (
	"os"
	"testing"
	"time"
)

func TestFromEnvStrictRejectsBadDuration(t *testing.T) {
	os.Setenv("AUTO_INTERLOCK_AFTER", "not-a-duration")
	defer os.Unsetenv("AUTO_INTERLOCK_AFTER")
	if _, err := FromEnvStrict(); err == nil {
		t.Fatal("expected FromEnvStrict to reject invalid duration")
	}
}

func TestFromEnvStrictAppliesValidOverrides(t *testing.T) {
	os.Setenv("PORT", "9090")
	os.Setenv("LOG_LEVEL", "debug")
	os.Setenv("AUTO_INTERLOCK_AFTER", "7m")
	defer func() {
		os.Unsetenv("PORT")
		os.Unsetenv("LOG_LEVEL")
		os.Unsetenv("AUTO_INTERLOCK_AFTER")
	}()
	cfg, err := FromEnvStrict()
	if err != nil {
		t.Fatalf("FromEnvStrict: %v", err)
	}
	if cfg.Port != "9090" || cfg.LogLevel != "debug" || cfg.AutoInterlockAfter != 7*time.Minute {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}
