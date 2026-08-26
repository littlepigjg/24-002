package handler

import (
	"net/http"

	"logalert/internal/service"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// SchedulerHandler handles HTTP requests for scheduler operations.
type SchedulerHandler struct {
	scheduler service.Scheduler
	logger    logger.Logger
}

// NewSchedulerHandler creates a new SchedulerHandler.
func NewSchedulerHandler(s service.Scheduler, log logger.Logger) *SchedulerHandler {
	return &SchedulerHandler{
		scheduler: s,
		logger:    log.WithField("handler", "scheduler"),
	}
}

// GetStatus handles GET /api/scheduler/status
func (h *SchedulerHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	status := h.scheduler.GetStatus()
	response.Success(status).Write(w)
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
