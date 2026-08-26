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
	config *config.Config
	logger logger.Logger
}

// NewConfigHandler creates a new ConfigHandler.
func NewConfigHandler(cfg *config.Config, log logger.Logger) *ConfigHandler {
	return &ConfigHandler{
		config: cfg,
		logger: log.WithField("handler", "config"),
	}
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

	// Apply updates
	if server, ok := updates["server"].(map[string]interface{}); ok {
		if host, ok := server["host"].(string); ok {
			h.config.Server.Host = host
		}
		if port, ok := server["port"].(float64); ok {
			h.config.Server.Port = int(port)
		}
	}

	if logging, ok := updates["logging"].(map[string]interface{}); ok {
		if level, ok := logging["level"].(string); ok {
			h.config.Logging.Level = level
		}
	}

	// Validate
	if err := h.config.Validate(); err != nil {
		response.Error(400, err.Error()).Write(w)
		return
	}

	h.logger.Info("configuration updated")
	response.Success(nil).Write(w)
}

// ReloadConfig handles POST /api/config/reload
func (h *ConfigHandler) ReloadConfig(w http.ResponseWriter, r *http.Request) {
	// In a real application, this would reload from file
	h.logger.Info("configuration reload requested")
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
