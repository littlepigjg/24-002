package handler

import (
	"net/http"
	"strings"
	"time"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// StatsHandler handles HTTP requests for statistics operations.
type StatsHandler struct {
	service service.StatsService
	logger  logger.Logger
}

// NewStatsHandler creates a new StatsHandler.
func NewStatsHandler(s service.StatsService, log logger.Logger) *StatsHandler {
	return &StatsHandler{
		service: s,
		logger:  log.WithField("handler", "stats"),
	}
}

// GetStatistics handles GET /api/stats
func (h *StatsHandler) GetStatistics(w http.ResponseWriter, r *http.Request) {
	req := model.DefaultStatsRequest()

	// Parse query parameters
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			req.StartTime = t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			req.EndTime = t
		}
	}
	if svc := r.URL.Query().Get("service"); svc != "" {
		req.Service = svc
	}
	if src := r.URL.Query().Get("source"); src != "" {
		req.Source = src
	}

	stats, err := h.service.GetLogStatistics(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to get statistics", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(stats).Write(w)
}

// GetHourlyBreakdown handles GET /api/stats/hourly
func (h *StatsHandler) GetHourlyBreakdown(w http.ResponseWriter, r *http.Request) {
	req := model.DefaultStatsRequest()

	// Parse query parameters
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			req.StartTime = t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			req.EndTime = t
		}
	}

	breakdown, err := h.service.GetHourlyBreakdown(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to get hourly breakdown", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(breakdown).Write(w)
}

// GetErrorRateTrend handles GET /api/stats/error-rate
func (h *StatsHandler) GetErrorRateTrend(w http.ResponseWriter, r *http.Request) {
	req := model.DefaultStatsRequest()

	// Parse query parameters
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			req.StartTime = t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			req.EndTime = t
		}
	}

	trend, err := h.service.GetErrorRateTrend(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to get error rate trend", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(trend).Write(w)
}

// GetSourceCount handles GET /api/stats/sources
func (h *StatsHandler) GetSourceCount(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			from = t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			now = t
		}
	}

	counts, err := h.service.GetSourceLogCount(r.Context(), from, now)
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(counts).Write(w)
}

// GetLevelDistribution handles GET /api/stats/levels
func (h *StatsHandler) GetLevelDistribution(w http.ResponseWriter, r *http.Request) {
	now := time.Now()
	from := now.Add(-24 * time.Hour)

	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			from = t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			now = t
		}
	}

	dist, err := h.service.GetLevelDistribution(r.Context(), from, now)
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(dist).Write(w)
}

// RegisterRoutes registers stats handler routes on the mux.
func (h *StatsHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/stats", h.GetStatistics)
	mux.HandleFunc("/api/stats/hourly", h.GetHourlyBreakdown)
	mux.HandleFunc("/api/stats/error-rate", h.GetErrorRateTrend)
	mux.HandleFunc("/api/stats/sources", h.GetSourceCount)
	mux.HandleFunc("/api/stats/levels", h.GetLevelDistribution)

	// Also handle /api/stats/ with trailing slash
	mux.HandleFunc("/api/stats/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimSuffix(r.URL.Path, "/")
		switch {
		case path == "/api/stats":
			h.GetStatistics(w, r)
		case strings.HasSuffix(path, "/hourly"):
			h.GetHourlyBreakdown(w, r)
		case strings.HasSuffix(path, "/error-rate"):
			h.GetErrorRateTrend(w, r)
		case strings.HasSuffix(path, "/sources"):
			h.GetSourceCount(w, r)
		case strings.HasSuffix(path, "/levels"):
			h.GetLevelDistribution(w, r)
		default:
			response.Error(404, "not found").Write(w)
		}
	})
}
