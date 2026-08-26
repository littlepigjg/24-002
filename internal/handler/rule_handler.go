package handler

import (
	"fmt"
	"net/http"
	"strings"

	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/pkg/jsonutil"
	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// RuleHandler handles HTTP requests for rule operations.
type RuleHandler struct {
	service service.RuleService
	logger  logger.Logger
}

// NewRuleHandler creates a new RuleHandler.
func NewRuleHandler(s service.RuleService, log logger.Logger) *RuleHandler {
	return &RuleHandler{
		service: s,
		logger:  log.WithField("handler", "rule"),
	}
}

// CreateRule handles POST /api/rules
func (h *RuleHandler) CreateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	var req model.CreateRuleRequest
	if err := jsonutil.ReadJSON(r, &req); err != nil {
		response.Error(400, fmt.Sprintf("invalid request: %v", err)).Write(w)
		return
	}

	errors := req.Validate()
	if len(errors) > 0 {
		response.Error(400, strings.Join(errors, "; ")).Write(w)
		return
	}

	rule, err := h.service.CreateRule(r.Context(), &req)
	if err != nil {
		h.logger.Error("failed to create rule", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(rule).Write(w)
}

// GetRule handles GET /api/rules/{id}
func (h *RuleHandler) GetRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/rules/")
	if id == "" {
		response.Error(400, "rule ID is required").Write(w)
		return
	}

	rule, err := h.service.GetRule(r.Context(), id)
	if err != nil {
		response.Error(404, err.Error()).Write(w)
		return
	}

	response.Success(rule).Write(w)
}

// UpdateRule handles PUT /api/rules/{id}
func (h *RuleHandler) UpdateRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/rules/")
	if id == "" {
		response.Error(400, "rule ID is required").Write(w)
		return
	}

	var req model.UpdateRuleRequest
	if err := jsonutil.ReadJSON(r, &req); err != nil {
		response.Error(400, fmt.Sprintf("invalid request: %v", err)).Write(w)
		return
	}

	rule, err := h.service.UpdateRule(r.Context(), id, &req)
	if err != nil {
		h.logger.Error("failed to update rule", "error", err)
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(rule).Write(w)
}

// DeleteRule handles DELETE /api/rules/{id}
func (h *RuleHandler) DeleteRule(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodDelete {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/rules/")
	if id == "" {
		response.Error(400, "rule ID is required").Write(w)
		return
	}

	if err := h.service.DeleteRule(r.Context(), id); err != nil {
		response.Error(404, err.Error()).Write(w)
		return
	}

	response.Success(nil).Write(w)
}

// ListRules handles GET /api/rules
func (h *RuleHandler) ListRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.ListRules(r.Context())
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}
	response.Success(rules).Write(w)
}

// ListActiveRules handles GET /api/rules/active
func (h *RuleHandler) ListActiveRules(w http.ResponseWriter, r *http.Request) {
	rules, err := h.service.ListActiveRules(r.Context())
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}
	response.Success(rules).Write(w)
}

// ToggleRuleStatus handles PUT /api/rules/{id}/status
func (h *RuleHandler) ToggleRuleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPut {
		response.Error(405, "method not allowed").Write(w)
		return
	}

	id := extractIDFromPath(r.URL.Path, "/api/rules/")
	if id == "" {
		response.Error(400, "rule ID is required").Write(w)
		return
	}

	// Extract status from path suffix
	statusPath := strings.TrimPrefix(r.URL.Path, "/api/rules/")
	statusPath = strings.TrimPrefix(statusPath, id+"/")
	statusPath = strings.TrimSuffix(statusPath, "/status")

	var req model.ToggleRuleRequest
	if err := jsonutil.ReadJSON(r, &req); err != nil {
		// Try to parse from body
		response.Error(400, fmt.Sprintf("invalid request: %v", err)).Write(w)
		return
	}

	if req.Status == "" {
		// Try to infer from path
		if statusPath != "" {
			req.Status = model.RuleStatus(statusPath)
		}
	}

	if req.Status != model.RuleActive && req.Status != model.RulePaused && req.Status != model.RuleArchived {
		response.Error(400, "invalid status").Write(w)
		return
	}

	rule, err := h.service.ToggleRuleStatus(r.Context(), id, req.Status)
	if err != nil {
		response.Error(500, err.Error()).Write(w)
		return
	}

	response.Success(rule).Write(w)
}

// RegisterRoutes registers rule handler routes on the mux.
func (h *RuleHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/rules", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			h.CreateRule(w, r)
		case http.MethodGet:
			h.ListRules(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
	})
	mux.HandleFunc("/api/rules/active", h.ListActiveRules)
	mux.HandleFunc("/api/rules/", func(w http.ResponseWriter, r *http.Request) {
		id := extractIDFromPath(r.URL.Path, "/api/rules/")
		// Check if this is a status toggle request
		if strings.HasSuffix(r.URL.Path, "/status") {
			h.ToggleRuleStatus(w, r)
			return
		}
		switch r.Method {
		case http.MethodGet:
			h.GetRule(w, r)
		case http.MethodPut:
			h.UpdateRule(w, r)
		case http.MethodDelete:
			h.DeleteRule(w, r)
		default:
			response.Error(405, "method not allowed").Write(w)
		}
		_ = id
	})
}
