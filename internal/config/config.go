package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	DefaultPollInterval = 10 * time.Minute
	AppName             = "emir-agent"
)

// Config holds runtime configuration loaded from flags or env vars.
type Config struct {
	CoreURL     string
	StatePath   string
	PollInterval time.Duration
}

// Load builds a Config from environment variables and defaults.
func Load() (*Config, error) {
	coreURL := os.Getenv("EMIR_CORE_URL")
	if coreURL == "" {
		coreURL = "http://localhost:8000"
	}

	statePath := os.Getenv("EMIR_STATE_PATH")
	if statePath == "" {
		configDir, err := os.UserConfigDir()
		if err != nil {
			return nil, fmt.Errorf("get user config dir: %w", err)
		}
		statePath = filepath.Join(configDir, AppName, "state.json")
	}

	return &Config{
		CoreURL:      coreURL,
		StatePath:    statePath,
		PollInterval: DefaultPollInterval,
	}, nil
}

// StateDir returns the directory where state.json lives.
func (c *Config) StateDir() string {
	return filepath.Dir(c.StatePath)
}
