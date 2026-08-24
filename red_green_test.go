package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"logalert/internal/config"
	"logalert/internal/handler"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	initialConfig := map[string]interface{}{
		"server": map[string]interface{}{
			"host":               "0.0.0.0",
			"port":               8080,
			"read_timeout":       30,
			"write_timeout":      30,
			"idle_timeout":       120,
			"max_request_body_size": 10485760,
		},
		"storage": map[string]interface{}{
			"max_log_entries":   100000,
			"max_alert_records": 10000,
			"persist_to_file":   false,
			"data_dir":          "./data",
		},
		"alert": map[string]interface{}{
			"default_scan_interval": 60,
			"max_rules_per_source":   100,
			"alert_history_size":    1000,
			"cooldown_duration":      300,
		},
		"logging": map[string]interface{}{
			"level":           "INFO",
			"output":          "stdout",
			"max_field_length": 1024,
		},
		"scheduler": map[string]interface{}{
			"scan_interval":        30,
			"max_concurrent_scans": 4,
			"enable_auto_scan":     true,
		},
	}

	configBytes, err := json.MarshalIndent(initialConfig, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal config: %v", err)
	}

	if err := os.WriteFile(configPath, configBytes, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}

	cfg, err := config.LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	log := logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter())
	logLevel := logger.ParseLogLevel(cfg.Logging.Level)
	log.SetLevel(logLevel)

	configHandler := handler.NewConfigHandler(cfg, log)
	configHandler.SetConfigPath(configPath)

	// Verify initial level is INFO before applying update
	initialLevel := configHandler.GetEffectiveLevel()
	if initialLevel != logger.LogLevelInfo {
		t.Fatalf("expected initial level INFO, got %v", initialLevel)
	}

	updatePayload := map[string]interface{}{
		"logging": map[string]interface{}{
			"level": "DEBUG",
		},
	}

	body, err := json.Marshal(updatePayload)
	if err != nil {
		t.Fatalf("failed to marshal update payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPut, "/api/config", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	configHandler.UpdateConfig(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rr.Code, rr.Body.String())
	}

	actualLevel := configHandler.GetEffectiveLevel()
	expectedLevel := logger.LogLevelDebug

	if actualLevel == expectedLevel {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Printf("RED (红灯，缺陷未修复)\n")
		t.Errorf("expected log level %v after config update, got %v", expectedLevel, actualLevel)
	}
}
