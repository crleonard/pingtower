package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr         string
	DataFile           string
	RequestUserAgent   string
	DefaultInterval    time.Duration
	DefaultTimeout     time.Duration
	MaxHistoryPerCheck int
}

func Load() Config {
	return Config{
		ListenAddr:         envOrDefault("PINGTOWER_ADDR", ":8080"),
		DataFile:           envOrDefault("PINGTOWER_DATA_FILE", "data/pingtower.json"),
		RequestUserAgent:   envOrDefault("PINGTOWER_USER_AGENT", "pingtower/1.0"),
		DefaultInterval:    durationEnv("PINGTOWER_DEFAULT_INTERVAL", 60*time.Second),
		DefaultTimeout:     durationEnv("PINGTOWER_DEFAULT_TIMEOUT", 10*time.Second),
		MaxHistoryPerCheck: intEnv("PINGTOWER_MAX_HISTORY", 100),
	}
}

func envOrDefault(key, fallback string) string {
	value := os.Getenv(key)
	if value == "" {
		return fallback
	}
	return value
}

func durationEnv(key string, fallback time.Duration) time.Duration {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := time.ParseDuration(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}

func intEnv(key string, fallback int) int {
	raw := os.Getenv(key)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return fallback
	}
	return value
}
