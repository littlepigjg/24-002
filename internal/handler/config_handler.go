// Package handler implements HTTP request handlers for the API.
package handler

import (
	"net/http"
	"time"

	"logalert/internal/config"
	"logalert/pkg/jsonutil"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// ConfigHandler handles HTTP requests for configuration operations.
type ConfigHandler struct {
	config     *config.Config
	logger     logger.Logger
	configPath string
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(cfg *config.Config, log logger.Logger) *ConfigHandler {
	return &ConfigHandler{
		config: cfg,
		logger: log.WithField("handler", "config"),
	}
}

// SetConfigPath sets the file path used for saving and reloading configuration.
func (h *ConfigHandler) SetConfigPath(path string) {
	h.configPath = path
}

// GetConfigPath returns the current configuration file path.
func (h *ConfigHandler) GetConfigPath() string {
	return h.configPath
}

// GetEffectiveLevel returns the current effective log level from the handler's logger.
func (h *ConfigHandler) GetEffectiveLevel() logger.LogLevel {
	if h.logger == nil {
		return logger.LogLevelInfo
	}
	return h.logger.GetLevel()
}

// GetConfig handles GET /api/config
func (h *ConfigHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	if h.config == nil {
		response.Error(500, "configuration not available").Write(w)
		return
	}

	configJSON, err := h.config.ToJSON()
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	var configMap map[string]interface{}
	if err := jsonutil.DecodeJSONBytes([]byte(configJSON), &configMap); err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(configMap).Write(w)
}

// UpdateConfig handles PUT /api/config
func (h *ConfigHandler) UpdateConfig(w http.ResponseWriter, r *http.Request) {
	if h.config == nil {
		response.Error(500, "configuration not available").Write(w)
		return
	}

	var updates map[string]interface{}
	if err := jsonutil.ReadJSON(r, &updates); err != nil {
		response.Error(400, err.Error()).Write(w)
		return
	}

	if server, ok := updates["server"].(map[string]interface{}); ok {
		if host, ok := server["host"].(string); ok {
			h.config.Server.Host = host
		}
		if port, ok := server["port"].(float64); ok {
			h.config.Server.Port = int(port)
		}
		if readTimeout, ok := server["read_timeout"].(float64); ok {
			h.config.Server.ReadTimeout = time.Duration(readTimeout)
		}
		if writeTimeout, ok := server["write_timeout"].(float64); ok {
			h.config.Server.WriteTimeout = time.Duration(writeTimeout)
		}
		if idleTimeout, ok := server["idle_timeout"].(float64); ok {
			h.config.Server.IdleTimeout = time.Duration(idleTimeout)
		}
		if maxSize, ok := server["max_request_body_size"].(float64); ok {
			h.config.Server.MaxRequestBodySize = int64(maxSize)
		}
	}

	if storage, ok := updates["storage"].(map[string]interface{}); ok {
		if maxEntries, ok := storage["max_log_entries"].(float64); ok {
			h.config.Storage.MaxLogEntries = int(maxEntries)
		}
		if maxAlerts, ok := storage["max_alert_records"].(float64); ok {
			h.config.Storage.MaxAlertRecords = int(maxAlerts)
		}
		if persist, ok := storage["persist_to_file"].(bool); ok {
			h.config.Storage.PersistToFile = persist
		}
		if dataDir, ok := storage["data_dir"].(string); ok {
			h.config.Storage.DataDir = dataDir
		}
	}

	if alert, ok := updates["alert"].(map[string]interface{}); ok {
		if scanInterval, ok := alert["default_scan_interval"].(float64); ok {
			h.config.Alert.DefaultScanInterval = time.Duration(scanInterval)
		}
		if maxRules, ok := alert["max_rules_per_source"].(float64); ok {
			h.config.Alert.MaxRulesPerSource = int(maxRules)
		}
		if alertHistory, ok := alert["alert_history_size"].(float64); ok {
			h.config.Alert.AlertHistorySize = int(alertHistory)
		}
		if cooldown, ok := alert["cooldown_duration"].(float64); ok {
			h.config.Alert.CooldownDuration = time.Duration(cooldown)
		}
	}

	if logging, ok := updates["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			h.config.Logging.Level = level
		}
		if output, ok := logging["output"].(string); ok {
			h.config.Logging.Output = output
		}
		if maxFieldLen, ok := logging["max_field_length"].(float64); ok {
			h.config.Logging.MaxFieldLength = int(maxFieldLen)
		}
	}

	if scheduler, ok := updates["scheduler"].(map[string]interface{}); ok {
		if scanInterval, ok := scheduler["scan_interval"].(float64); ok {
			h.config.Scheduler.ScanInterval = time.Duration(scanInterval)
		}
		if maxConcurrent, ok := scheduler["max_concurrent_scans"].(float64); ok {
			h.config.Scheduler.MaxConcurrentScans = int(maxConcurrent)
		}
		if enableAuto, ok := scheduler["enable_auto_scan"].(bool); ok {
			h.config.Scheduler.EnableAutoScan = enableAuto
		}
	}

	if err := h.config.Validate(); err != nil {
		response.Error(400, err.Error()).Write(w)
		return
	}

	if h.configPath != "" {
		if err := h.config.SaveToFile(h.configPath); err != nil {
			h.logger.Error("failed to save config", "error", err)
			response.Error(500, err.Error()).Write(w)
			return
		}

		if err := h.config.ReloadFromFile(h.configPath); err != nil {
			h.logger.Error("failed to reload config", "error", err)
			response.Error(500, err.Error()).Write(w)
			return
		}

		h.syncLoggerLevel()
	}

	h.logger.Info("configuration updated")
	response.Success(nil).Write(w)
}

// syncLoggerLevel updates the logger's level based on the current configuration.
func (h *ConfigHandler) syncLoggerLevel() {
	if h.config == nil || h.logger == nil {
		return
	}
	level := logger.ParseLogLevel(h.config.Logging.Level)
	h.logger.SetLevel(level)
}

// ReloadConfig handles POST /api/config/reload
func (h *ConfigHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	if h.configPath == "" {
		response.Error(400, "no config file path configured").Write(w)
		return
	}

	if err := h.config.ReloadFromFile(h.configPath); err != nil {
		h.logger.Error("configuration reload failed", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	h.syncLoggerLevel()

	h.logger.Info("configuration reloaded")
	response.SuccessMsg("configuration reloaded", time.Now()).Write(w)
}

// RegisterRoutes registers config handler routes on the mux.
func (h *ConfigHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/config", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetConfig(w, r)
		case http.MethodPut:
			h.UpdateConfig(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
	})
	mux.HandleFunc("/api/config/reload", func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			h.ReloadConfig(w, r)
		} else {
			response.Error(405, "method not allowed").Write(w)
		}
	})
}
