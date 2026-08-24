// Package config provides application configuration management.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

// Config holds all application configuration.
type Config struct {
	// Server configuration
	Server ServerConfig `json:"server"`
	// Storage configuration
	Storage StorageConfig `json:"storage"`
	// Alert configuration
	Alert AlertConfig `json:"alert"`
	// Logging configuration
	Logging LoggingConfig `json:"logging"`
	// Scheduler configuration
	Scheduler SchedulerConfig `json:"scheduler"`
}

// ServerConfig holds HTTP server configuration.
type ServerConfig struct {
	// Host is the host address to bind to.
	Host string `json:"host"`
	// Port is the port to listen on.
	Port int `json:"port"`
	// ReadTimeout is the maximum duration for reading the request.
	ReadTimeout time.Duration `json:"read_timeout"`
	// WriteTimeout is the maximum duration for writing the response.
	WriteTimeout time.Duration `json:"write_timeout"`
	// IdleTimeout is the maximum duration for idle connections.
	IdleTimeout time.Duration `json:"idle_timeout"`
	// MaxRequestBodySize is the maximum request body size in bytes.
	MaxRequestBodySize int64 `json:"max_request_body_size"`
	// ShutdownTimeout is the timeout for graceful shutdown.
	ShutdownTimeout time.Duration `json:"shutdown_timeout"`
}

// StorageConfig holds storage configuration.
type StorageConfig struct {
	// MaxLogEntries is the maximum number of log entries to keep in memory.
	MaxLogEntries int `json:"max_log_entries"`
	// MaxAlertRecords is the maximum number of alert records to keep.
	MaxAlertRecords int `json:"max_alert_records"`
	// PersistToFile enables file-based persistence.
	PersistToFile bool `json:"persist_to_file"`
	// DataDir is the directory for data persistence.
	DataDir string `json:"data_dir"`
}

// AlertConfig holds alert rule configuration.
type AlertConfig struct {
	// DefaultScanInterval is the default interval between rule scans.
	DefaultScanInterval time.Duration `json:"default_scan_interval"`
	// MaxRulesPerSource is the maximum number of rules per source.
	MaxRulesPerSource int `json:"max_rules_per_source"`
	// AlertHistorySize is the number of alert events to keep per rule.
	AlertHistorySize int `json:"alert_history_size"`
	// CooldownDuration is the minimum time between alerts for the same rule.
	CooldownDuration time.Duration `json:"cooldown_duration"`
}

// LoggingConfig holds logging configuration.
type LoggingConfig struct {
	// Level is the minimum log level to output.
	Level string `json:"level"`
	// Output is the output destination ("stdout" or file path).
	Output string `json:"output"`
	// MaxFieldLength is the maximum length of a log field value.
	MaxFieldLength int `json:"max_field_length"`
}

// SchedulerConfig holds scheduler configuration.
type SchedulerConfig struct {
	// ScanInterval is the interval between rule scans.
	ScanInterval time.Duration `json:"scan_interval"`
	// MaxConcurrentScans is the maximum number of concurrent rule scans.
	MaxConcurrentScans int `json:"max_concurrent_scans"`
	// EnableAutoScan enables automatic rule scanning.
	EnableAutoScan bool `json:"enable_auto_scan"`
}

// Default returns a Config with sensible defaults.
func Default() *Config {
	return DefaultConfig()
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host:               "0.0.0.0",
			Port:               8080,
			ReadTimeout:        30 * time.Second,
			WriteTimeout:       30 * time.Second,
			IdleTimeout:        120 * time.Second,
			MaxRequestBodySize: 10 * 1024 * 1024, // 10MB
			ShutdownTimeout:    15 * time.Second,
		},
		Storage: StorageConfig{
			MaxLogEntries:    100000,
			MaxAlertRecords:  10000,
			PersistToFile:     false,
			DataDir:          "./data",
		},
		Alert: AlertConfig{
			DefaultScanInterval: 1 * time.Minute,
			MaxRulesPerSource:   100,
			AlertHistorySize:    1000,
			CooldownDuration:    5 * time.Minute,
		},
		Logging: LoggingConfig{
			Level:          "INFO",
			Output:         "stdout",
			MaxFieldLength: 1024,
		},
		Scheduler: SchedulerConfig{
			ScanInterval:      30 * time.Second,
			MaxConcurrentScans: 4,
			EnableAutoScan:    true,
		},
	}
}

// LoadFromFile loads configuration from a JSON file.
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return cfg, nil
}

// Validate checks that the configuration values are valid.
func (c *Config) Validate() error {
	if c.Server.Port <= 0 || c.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", c.Server.Port)
	}
	if c.Server.ReadTimeout <= 0 {
		return fmt.Errorf("read timeout must be positive")
	}
	if c.Server.WriteTimeout <= 0 {
		return fmt.Errorf("write timeout must be positive")
	}
	if c.Server.MaxRequestBodySize <= 0 {
		return fmt.Errorf("max request body size must be positive")
	}
	if c.Storage.MaxLogEntries <= 0 {
		return fmt.Errorf("max log entries must be positive")
	}
	if c.Alert.DefaultScanInterval <= 0 {
		return fmt.Errorf("default scan interval must be positive")
	}
	if c.Scheduler.ScanInterval <= 0 {
		return fmt.Errorf("scan interval must be positive")
	}
	return nil
}

// ToJSON serializes the configuration to JSON.
func (c *Config) ToJSON() (string, error) {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal config: %w", err)
	}
	return string(data), nil
}

// URLFilePath sets the file path for URL persistence.
func (s *StorageConfig) URLFilePath(path string) *StorageConfig {
	s.DataDir = path
	return s
}

// LogFilePath sets the file path for log persistence.
func (s *StorageConfig) LogFilePath(path string) *StorageConfig {
	s.DataDir = path
	return s
}

// SyncInterval sets the sync interval for storage operations.
func (s *StorageConfig) SyncInterval(d time.Duration) *StorageConfig {
	return s
}

// FlushOnWrite sets whether to flush on write.
func (s *StorageConfig) FlushOnWrite(b bool) *StorageConfig {
	return s
}

// SaveToFile saves the configuration to a JSON file.
func (c *Config) SaveToFile(path string) error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
