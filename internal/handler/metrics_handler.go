// Package handler implements HTTP request handlers for the API.
package handler

import (
	"net/http"
	"time"

	"logalert/pkg/logger"
	"logalert/pkg/metrics"
	"logalert/pkg/response"
)

// MetricsHandler handles HTTP requests for metrics/monitoring operations.
type MetricsHandler struct {
	logger  logger.Logger
	metrics *metrics.Metrics
}

// NewMetricsHandler creates a new MetricsHandler.
func NewMetricsHandler(log logger.Logger) *MetricsHandler {
	return &MetricsHandler{
		logger:  log.WithField("handler", "metrics"),
		metrics: metrics.Global(),
	}
}

// GetMetrics handles GET /api/metrics
func (h *MetricsHandler) GetMetrics(w http.ResponseWriter, r *http.Request) {
	snapshot := h.metrics.Snapshot()
	snapshot["server_time"] = time.Now()
	response.Success(snapshot).Write(w)
}

// ResetMetrics handles POST /api/metrics/reset
func (h *MetricsHandler) ResetMetrics(w http.ResponseWriter, r *http.Request) {
	h.metrics.Reset()
	response.SuccessMsg("metrics reset", time.Now()).Write(w)
}

// RegisterRoutes registers metrics handler routes on the mux.
func (h *MetricsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/metrics", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetMetrics(w, r)
		case http.MethodPost:
			h.ResetMetrics(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
	})
}
