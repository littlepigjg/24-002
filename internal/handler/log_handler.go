// Package handler implements HTTP request handlers for the API.
package handler

import (
	"encoding/json"
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

// LogHandler handles HTTP requests for log operations.
type LogHandler struct {
	service service.LogService
	logger  logger.Logger
}

// NewLogHandler creates a new LogHandler.
func NewLogHandler(s service.LogService, log logger.Logger) *LogHandler {
	return &LogHandler{
		service: s,
		logger:  log.WithField("handler", "log"),
	}
}

// CreateLog handles POST /api/logs
func (h *LogHandler) CreateLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	var req model.CreateLogRequest
	if err := jsonutil.ReadJSON(r, &req); err != nil {
		response.Error(400, fmt.Sprintf("invalid request: %v", err)).Write(w)
		return
	}

	req.Level = normalizeLevel(req.Level)

	errors := req.Validate()
	if len(errors) > 0 {
		response.Error(400, strings.Join(errors, "; ")).Write(w)
		return
	}

	entry, err := h.service.CreateLog(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create log", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(entry).Write(w)
}

func normalizeLevel(level model.LogLevel) model.LogLevel {
	lower := strings.ToLower(string(level))
	switch lower {
	case "info":
		return model.LogLevel(lower)
	case "warn":
		return model.LogLevel(lower)
	case "error":
		return model.LogLevel(lower)
	case "debug":
		return model.LogLevel(lower)
	case "fatal":
		return model.LogLevel(lower)
	default:
		return model.LogLevel(lower)
	}
}

// CreateLogs handles POST /api/logs/batch
func (h *LogHandler) CreateLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	var reqs []*model.CreateLogRequest
	if err := jsonutil.ReadJSON(r, &reqs); err != nil {
		response.Error(400, fmt.Sprintf("invalid request: %v", err)).Write(w)
		return
	}

	// Validate all
	for _, req := range reqs {
		if errs := req.Validate(); len(errs) > 0 {
			response.Error(400, fmt.Sprintf("invalid request: %s", strings.Join(errs, "; "))).Write(w)
			return
		}
	}

	entries, err := h.service.CreateLogs(r.Context(), reqs)
	if err != nil {
		h.logger.Error("failed to create logs", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(entries).Write(w)
}

// GetLog handles GET /api/logs/{id}
func (h *LogHandler) GetLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/logs/")
	if id == "" {
		response.Error(400, "log ID is required").Write(w)
		return
	}

	entry, err := h.service.GetLog(r.Context(), id)
	if err != nil {
		response.Error(404, err.Error()).Write(w)
		return
	}

	response.Success(entry).Write(w)
}

// QueryLogs handles GET /api/logs
func (h *LogHandler) QueryLogs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	req := model.DefaultQueryLogsRequest()

	// Parse query parameters
	if levelsStr := r.URL.Query().Get("levels"); levelsStr != "" {
		levels := strings.Split(levelsStr, ",")
		for _, l := range levels {
			req.Levels = append(req.Levels, model.LogLevel(strings.TrimSpace(l)))
		}
	}
	if sourcesStr := r.URL.Query().Get("sources"); sourcesStr != "" {
		req.Sources = strings.Split(sourcesStr, ",")
	}
	if svc := r.URL.Query().Get("service"); svc != "" {
		req.Service = svc
	}
	if keywordsStr := r.URL.Query().Get("keywords"); keywordsStr != "" {
		req.Keywords = strings.Split(keywordsStr, ",")
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

	// Validate
	errors := req.Validate()
	if len(errors) > 0 {
		response.Error(400, strings.Join(errors, "; ")).Write(w)
		return
	}

	results, count, err := h.service.QueryLogs(r.Context(), req)
	if err != nil {
		h.logger.Error("failed to query logs", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Paginated(results, count, req.Offset/req.Limit+1, req.Limit).Write(w)
}

// DeleteLog handles DELETE /api/logs/{id}
func (h *LogHandler) DeleteLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/logs/")
	if id == "" {
		response.Error(400, "log ID is required").Write(w)
		return
	}

	if err := h.service.DeleteLog(r.Context(), id); err != nil {
		response.Error(404, err.Error()).Write(w)
		return
	}

	response.Success(nil).Write(w)
}

// ListSources handles GET /api/logs/sources
func (h *LogHandler) ListSources(w http.ResponseWriter, r *http.Request) {
	sources, err := h.service.ListSources(r.Context())
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}
	response.Success(sources).Write(w)
}

// ListServices handles GET /api/logs/services
func (h *LogHandler) ListServices(w http.ResponseWriter, r *http.Request) {
	services, err := h.service.ListServices(r.Context())
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}
	response.Success(services).Write(w)
}

// RegisterRoutes registers log handler routes on the mux.
func (h *LogHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/logs", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateLog(w, r)
		case http.MethodGet:
			h.QueryLogs(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
	})
	mux.HandleFunc("/api/logs/batch", h.CreateLogs)
	mux.HandleFunc("/api/logs/sources", h.ListSources)
	mux.HandleFunc("/api/logs/services", h.ListServices)
	mux.HandleFunc("/api/logs/", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			h.GetLog(w, r)
		case http.MethodDelete:
			h.DeleteLog(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
	})
}

// extractIDFromPath extracts the ID from a URL path after a prefix.
func extractIDFromPath(path, prefix string) string {
	id := strings.TrimPrefix(path, prefix)
	id = strings.Trim(id, "/")
	return id
}

// Ensure unused imports don't cause errors
var _ = json.Marshal
