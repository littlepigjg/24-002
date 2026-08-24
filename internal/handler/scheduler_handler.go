package handler

import (
	"net/http"
	"sort"

	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// SchedulerHandler handles HTTP requests for scheduler operations.
type SchedulerHandler struct {
	scheduler   service.Scheduler
	logger      logger.Logger
	metricStore *HealthMetricStore
}

// NewSchedulerHandler creates a new SchedulerHandler.
func NewSchedulerHandler(s service.Scheduler, log logger.Logger, metricStore *HealthMetricStore) *SchedulerHandler {
	return &SchedulerHandler{
		scheduler:   s,
		logger:      log.WithField("handler", "scheduler"),
		metricStore: metricStore,
	}
}

// GetStatus handles GET /api/scheduler/status
func (h *SchedulerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scheduler.GetStatus()

	metrics := h.metricStore.GetMetrics()
	sortedMetrics := make([]HealthMetric, len(metrics))
	copy(sortedMetrics, metrics)
	sort.Slice(sortedMetrics, func(i, j int) bool {
		return sortedMetrics[i].Timestamp.After(sortedMetrics[j].Timestamp)
	})

	// When no metrics have been recorded yet (e.g. right after startup),
	// there is no latest sample to report. Default to 0 instead of indexing
	// into an empty slice and panicking.
	var latestValue float64
	if len(sortedMetrics) > 0 {
		latestValue = sortedMetrics[0].Value
	}
	statusInfo := map[string]interface{}{
		"status":        status,
		"latest_metric": latestValue,
	}

	response.Success(statusInfo).Write(w)
}

// ScanNow handles POST /api/scheduler/scan
func (h *SchedulerHandler) ScanNow(w http.ResponseWriter, r *http.Request) {
	if err := h.scheduler.ScanOnce(r.Context()); err != nil {
		h.logger.Error("scan failed", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}
	response.SuccessMsg("scan completed", h.scheduler.GetStatus()).Write(w)
}

// RegisterRoutes registers scheduler handler routes on the mux.
func (h *SchedulerHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/scheduler/status", h.GetStatus)
	mux.HandleFunc("/api/scheduler/scan", h.ScanNow)
}
