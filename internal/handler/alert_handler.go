package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/jsonutil"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// AlertHandler handles HTTP requests for alert operations.
type AlertHandler struct {
	service service.AlertService
	logger  logger.Logger
}

// NewAlertHandler creates a new AlertHandler.
func NewAlertHandler(s service.AlertService, log logger.Logger) *AlertHandler {
	return &AlertHandler{
		service: s,
		logger:  log.WithField("handler", "alert"),
	}
}

// GetAlert handles GET /api/alerts/{id}
func (h *AlertHandler) GetAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/alerts/")
	if id == "" {
		response.Error(400, "alert ID is required").Write(w)
		return
	}

	alert, err := h.service.GetAlert(r.Context(), id)
	if err != nil {
		response.Error(404, err.Error()).Write(w)
		return
	}

	response.Success(alert).Write(w)
}

// QueryAlerts handles GET /api/alerts
func (h *AlertHandler) QueryAlerts(w http.ResponseWriter, r *http.Request) {
	req := model.DefaultQueryAlertsRequest()

	// Parse query parameters
	if statusesStr := r.URL.Query().Get("statuses"); statusesStr != "" {
		statuses := strings.Split(statusesStr, ",")
		for _, s := range statuses {
			req.Statuses = append(req.Statuses, model.AlertStatus(strings.TrimSpace(s)))
		}
	}
	if severitiesStr := r.URL.Query().Get("severities"); severitiesStr != "" {
		severities := strings.Split(severitiesStr, ",")
		for _, s := range severities {
			req.Severities = append(req.Severities, model.Severity(strings.TrimSpace(s)))
		}
	}
	if source := r.URL.Query().Get("source"); source != "" {
		req.Source = source
	}
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			req.Limit = l
		}
	}
	if offsetStr := r.URL.Query().Get("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			req.Offset = o
		}
	}
	if startStr := r.URL.Query().Get("start_time"); startStr != "" {
		if t, err := time.Parse(time.RFC3339, startStr); err == nil {
			req.StartTime = &t
		}
	}
	if endStr := r.URL.Query().Get("end_time"); endStr != "" {
		if t, err := time.Parse(time.RFC3339, endStr); err == nil {
			req.EndTime = &t
		}
	}

	results, count, err := h.service.QueryAlerts(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to query alerts", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Paginated(results, count, req.Offset/req.Limit+1, req.Limit).Write(w)
}

// AcknowledgeAlert handles POST /api/alerts/{id}/acknowledge
func (h *AlertHandler) AcknowledgeAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/alerts/")
	if id == "" {
		response.Error(400, "alert ID is required").Write(w)
		return
	}

	var req model.AcknowledgeAlertRequest
	if err := jsonutil.ReadJSON(r, &req); err != nil {
		response.Error(400, fmt.Sprintf("invalid request: %v", err)).Write(w)
		return
	}

	if req.User == "" {
		response.Error(400, "user is required").Write(w)
		return
	}

	alert, err := h.service.AcknowledgeAlert(r.Context(), id, &req)
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(alert).Write(w)
}

// ResolveAlert handles POST /api/alerts/{id}/resolve
func (h *AlertHandler) ResolveAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/alerts/")
	if id == "" {
		response.Error(400, "alert ID is required").Write(w)
		return
	}

	alert, err := h.service.ResolveAlert(r.Context(), id)
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(alert).Write(w)
}

// DeleteAlert handles DELETE /api/alerts/{id}
func (h *AlertHandler) DeleteAlert(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/alerts/")
	if id == "" {
		response.Error(400, "alert ID is required").Write(w)
		return
	}

	if err := h.service.DeleteAlert(r.Context(), id); err != nil {
		response.Error(404, err.Error()).Write(w)
		return
	}

	response.Success(nil).Write(w)
}

// ListRecentAlerts handles GET /api/alerts/recent
func (h *AlertHandler) ListRecentAlerts(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	alerts, err := h.service.ListRecent(r.Context(), limit)
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(alerts).Write(w)
}

// RegisterRoutes registers alert handler routes on the mux.
func (h *AlertHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/alerts", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.QueryAlerts(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
	})
	mux.HandleFunc("/api/alerts/recent", h.ListRecentAlerts)
	mux.HandleFunc("/api/alerts/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch {
		case strings.HasSuffix(path, "/acknowledge"):
			h.AcknowledgeAlert(w, r)
		case strings.HasSuffix(path, "/resolve"):
			h.ResolveAlert(w, r)
		default:
			switch r.Method {
			case http.MethodGet:
				h.GetAlert(w, r)
			case http.MethodDelete:
				h.DeleteAlert(w, r)
			default:
				response.Error(405, "method not allowed").Write(w)
			}
		}
	})
}
