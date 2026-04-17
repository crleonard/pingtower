package config

import (
	"testing"
	"time"
)

func TestLoadUsesEnvironmentOverrides(t *testing.T) {
	t.Setenv("PINGTOWER_ADDR", ":9090")
	t.Setenv("PINGTOWER_DATA_FILE", "/tmp/pingtower.json")
	t.Setenv("PINGTOWER_USER_AGENT", "pingtower-test/1.0")
	t.Setenv("PINGTOWER_DEFAULT_INTERVAL", "45s")
	t.Setenv("PINGTOWER_DEFAULT_TIMEOUT", "7s")
	t.Setenv("PINGTOWER_MAX_HISTORY", "25")

	cfg := Load()

	if cfg.ListenAddr != ":9090" {
		t.Fatalf("ListenAddr = %q, want :9090", cfg.ListenAddr)
	}
	if cfg.DataFile != "/tmp/pingtower.json" {
		t.Fatalf("DataFile = %q, want /tmp/pingtower.json", cfg.DataFile)
	}
	if cfg.RequestUserAgent != "pingtower-test/1.0" {
		t.Fatalf("RequestUserAgent = %q, want pingtower-test/1.0", cfg.RequestUserAgent)
	}
	if cfg.DefaultInterval != 45*time.Second {
		t.Fatalf("DefaultInterval = %v, want 45s", cfg.DefaultInterval)
	}
	if cfg.DefaultTimeout != 7*time.Second {
		t.Fatalf("DefaultTimeout = %v, want 7s", cfg.DefaultTimeout)
	}
	if cfg.MaxHistoryPerCheck != 25 {
		t.Fatalf("MaxHistoryPerCheck = %d, want 25", cfg.MaxHistoryPerCheck)
	}
}

func TestLoadFallsBackForInvalidEnvironmentValues(t *testing.T) {
	t.Setenv("PINGTOWER_DEFAULT_INTERVAL", "bad")
	t.Setenv("PINGTOWER_DEFAULT_TIMEOUT", "0s")
	t.Setenv("PINGTOWER_MAX_HISTORY", "-1")

	cfg := Load()

	if cfg.DefaultInterval != 60*time.Second {
		t.Fatalf("DefaultInterval = %v, want 60s", cfg.DefaultInterval)
	}
	if cfg.DefaultTimeout != 10*time.Second {
		t.Fatalf("DefaultTimeout = %v, want 10s", cfg.DefaultTimeout)
	}
	if cfg.MaxHistoryPerCheck != 100 {
		t.Fatalf("MaxHistoryPerCheck = %d, want 100", cfg.MaxHistoryPerCheck)
	}
}
