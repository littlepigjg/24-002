// Package config provides application configuration management.
package config

import (
	"fmt"
	"os"
	"time"
)

// Load loads configuration from environment variables and file.
func Load(configPath string) (*Config, error) {
	cfg := DefaultConfig()

	// Load from file if exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			fileCfg, err := LoadFromFile(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load config file: %w", err)
			}
			cfg = fileCfg
		}
	}

	// Override with environment variables
	envOverrides(cfg)

	// Validate
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// envOverrides reads configuration from environment variables.
func envOverrides(cfg *Config) {
	if v := os.Getenv("SERVER_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("SERVER_PORT"); v != "" {
		if port, err := parseInt(v); err == nil {
			cfg.Server.Port = port
		}
	}
	if v := os.Getenv("READ_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.ReadTimeout = d
		}
	}
	if v := os.Getenv("WRITE_TIMEOUT"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Server.WriteTimeout = d
		}
	}
	if v := os.Getenv("STORAGE_MAX_LOG_ENTRIES"); v != "" {
		if n, err := parseInt(v); err == nil {
			cfg.Storage.MaxLogEntries = n
		}
	}
	if v := os.Getenv("STORAGE_MAX_ALERT_RECORDS"); v != "" {
		if n, err := parseInt(v); err == nil {
			cfg.Storage.MaxAlertRecords = n
		}
	}
	if v := os.Getenv("STORAGE_PERSIST"); v != "" {
		cfg.Storage.PersistToFile = v == "true" || v == "1"
	}
	if v := os.Getenv("STORAGE_DATA_DIR"); v != "" {
		cfg.Storage.DataDir = v
	}
	if v := os.Getenv("LOG_LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("LOG_OUTPUT"); v != "" {
		cfg.Logging.Output = v
	}
	if v := os.Getenv("SCHEDULER_INTERVAL"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Scheduler.ScanInterval = d
		}
	}
	if v := os.Getenv("SCHEDULER_ENABLE"); v != "" {
		cfg.Scheduler.EnableAutoScan = v == "true" || v == "1"
	}
	if v := os.Getenv("ALERT_COOLDOWN"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Alert.CooldownDuration = d
		}
	}
}

// parseInt parses a string to int.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}
