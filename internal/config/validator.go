// Package config provides application configuration management.
package config

import (
	"fmt"
	"strings"
	"time"
)

// Validate checks configuration values and returns a list of validation errors.
func (c *Config) ValidateAll() []string {
	var errors []string

	// Server validation
	if c.Server.Host == "" {
		errors = append(errors, "server host is required")
	}
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		errors = append(errors, fmt.Sprintf("invalid server port: %d", c.Server.Port))
	}
	if c.Server.ReadTimeout < 1*time.Second {
		errors = append(errors, "read timeout must be at least 1 second")
	}
	if c.Server.WriteTimeout < 1*time.Second {
		errors = append(errors, "write timeout must be at least 1 second")
	}
	if c.Server.IdleTimeout < 0 {
		errors = append(errors, "idle timeout must be non-negative")
	}
	if c.Server.MaxRequestBodySize < 1024 {
		errors = append(errors, "max request body size must be at least 1KB")
	}
	if c.Server.ShutdownTimeout < 1*time.Second {
		errors = append(errors, "shutdown timeout must be at least 1 second")
	}

	// Storage validation
	if c.Storage.MaxLogEntries < 100 {
		errors = append(errors, "max log entries must be at least 100")
	}
	if c.Storage.MaxAlertRecords < 10 {
		errors = append(errors, "max alert records must be at least 10")
	}

	// Alert validation
	if c.Alert.DefaultScanInterval < time.Second {
		errors = append(errors, "default scan interval must be at least 1 second")
	}
	if c.Alert.MaxRulesPerSource < 1 {
		errors = append(errors, "max rules per source must be at least 1")
	}
	if c.Alert.AlertHistorySize < 1 {
		errors = append(errors, "alert history size must be at least 1")
	}
	if c.Alert.CooldownDuration < 0 {
		errors = append(errors, "cooldown duration must be non-negative")
	}

	// Logging validation
	validLevels := map[string]bool{"DEBUG": true, "INFO": true, "WARN": true, "ERROR": true, "FATAL": true}
	if !validLevels[strings.ToUpper(c.Logging.Level)] {
		errors = append(errors, fmt.Sprintf("invalid log level: %s", c.Logging.Level))
	}

	// Scheduler validation
	if c.Scheduler.ScanInterval < time.Second {
		errors = append(errors, "scheduler scan interval must be at least 1 second")
	}
	if c.Scheduler.MaxConcurrentScans < 1 {
		errors = append(errors, "max concurrent scans must be at least 1")
	}

	return errors
}
