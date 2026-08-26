package model

import (
	"time"
)

// Request types for API endpoints.

// CreateLogRequest is the request to create a log entry.
type CreateLogRequest struct {
	// Timestamp is when the log was generated (client-side).
	Timestamp time.Time `json:"timestamp"`
	// Level is the severity level of the log.
	Level LogLevel `json:"level"`
	// Source is the service/component that generated the log.
	Source string `json:"source"`
	// Message is the log message.
	Message string `json:"message"`
	// Service is the service name.
	Service string `json:"service,omitempty"`
	// Tags are optional metadata tags.
	Tags map[string]string `json:"tags,omitempty"`
}

// Validate validates the CreateLogRequest.
func (r *CreateLogRequest) Validate() []string {
	var errors []string
	if r.Source == "" {
		errors = append(errors, "source is required")
	}
	if r.Level == "" {
		errors = append(errors, "level is required")
	}
	if r.Message == "" {
		errors = append(errors, "message is required")
	}
	if !isValidLevel(r.Level) {
		errors = append(errors, "invalid level")
	}
	if len(r.Message) > 10000 {
		errors = append(errors, "message too long")
	}
	return errors
}

// CreateRuleRequest is the request to create an alert rule.
type CreateRuleRequest struct {
	// Name is the rule name.
	Name string `json:"name"`
	// Description is the rule description.
	Description string `json:"description,omitempty"`
	// Condition defines the triggering condition.
	Condition RuleCondition `json:"condition"`
	// Window is the time window for evaluation.
	Window time.Duration `json:"window"`
	// Threshold is the triggering threshold.
	Threshold float64 `json:"threshold"`
	// Severity is the alert severity.
	Severity Severity `json:"severity"`
	// Cooldown is the minimum time between alerts.
	Cooldown time.Duration `json:"cooldown"`
}

// Validate validates the CreateRuleRequest.
func (r *CreateRuleRequest) Validate() []string {
	var errors []string
	if r.Name == "" {
		errors = append(errors, "name is required")
	}
	if r.Condition.Type == "" {
		errors = append(errors, "condition type is required")
	}
	if r.Window <= 0 {
		errors = append(errors, "window must be positive")
	}
	if r.Threshold <= 0 {
		errors = append(errors, "threshold must be positive")
	}
	return errors
}

// UpdateRuleRequest is the request to update an alert rule.
type UpdateRuleRequest struct {
	// Name is the new name (empty to keep unchanged).
	Name string `json:"name,omitempty"`
	// Description is the new description.
	Description string `json:"description,omitempty"`
	// Condition is the new condition.
	Condition *RuleCondition `json:"condition,omitempty"`
	// Window is the new window.
	Window *time.Duration `json:"window,omitempty"`
	// Threshold is the new threshold.
	Threshold *float64 `json:"threshold,omitempty"`
	// Severity is the new severity.
	Severity Severity `json:"severity,omitempty"`
}

// ToggleRuleRequest is the request to toggle a rule's status.
type ToggleRuleRequest struct {
	// Status is the desired status.
	Status RuleStatus `json:"status"`
}

// QueryLogsRequest is the request to query logs.
type QueryLogsRequest struct {
	// Levels to filter by.
	Levels []LogLevel `json:"levels,omitempty"`
	// Sources to filter by.
	Sources []string `json:"sources,omitempty"`
	// Service to filter by.
	Service string `json:"service,omitempty"`
	// Keywords to search for.
	Keywords []string `json:"keywords,omitempty"`
	// StartTime for the query range.
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime for the query range.
	EndTime *time.Time `json:"end_time,omitempty"`
	// Limit is the maximum number of results.
	Limit int `json:"limit"`
	// Offset is the number of results to skip.
	Offset int `json:"offset"`
	// SortBy is the field to sort by.
	SortBy string `json:"sort_by,omitempty"`
	// SortOrder is "asc" or "desc".
	SortOrder string `json:"sort_order,omitempty"`
}

// DefaultQueryLogsRequest returns a QueryLogsRequest with defaults.
func DefaultQueryLogsRequest() *QueryLogsRequest {
	return &QueryLogsRequest{
		Limit:     100,
		Offset:    0,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
}

// Validate validates the QueryLogsRequest.
func (r *QueryLogsRequest) Validate() []string {
	var errors []string
	if r.Limit <= 0 || r.Limit > 1000 {
		errors = append(errors, "limit must be between 1 and 1000")
	}
	if r.Offset < 0 {
		errors = append(errors, "offset must be non-negative")
	}
	return errors
}

// QueryAlertsRequest is the request to query alerts.
type QueryAlertsRequest struct {
	// Statuses to filter by.
	Statuses []AlertStatus `json:"statuses,omitempty"`
	// Severities to filter by.
	Severities []Severity `json:"severities,omitempty"`
	// RuleIDs to filter by.
	RuleIDs []string `json:"rule_ids,omitempty"`
	// Source to filter by.
	Source string `json:"source,omitempty"`
	// StartTime for the query range.
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime for the query range.
	EndTime *time.Time `json:"end_time,omitempty"`
	// Limit is the maximum number of results.
	Limit int `json:"limit"`
	// Offset is the number of results to skip.
	Offset int `json:"offset"`
}

// DefaultQueryAlertsRequest returns a QueryAlertsRequest with defaults.
func DefaultQueryAlertsRequest() *QueryAlertsRequest {
	return &QueryAlertsRequest{
		Limit:  50,
		Offset: 0,
	}
}

// AcknowledgeAlertRequest is the request to acknowledge an alert.
type AcknowledgeAlertRequest struct {
	// User is the user acknowledging the alert.
	User string `json:"user"`
}

// ResolveAlertRequest is the request to resolve an alert.
type ResolveAlertRequest struct {
	// Reason is the resolution reason.
	Reason string `json:"reason,omitempty"`
}

// StatsRequest is the request for statistics.
type StatsRequest struct {
	// StartTime is the start of the statistics period.
	StartTime time.Time `json:"start_time"`
	// EndTime is the end of the statistics period.
	EndTime time.Time `json:"end_time"`
	// GroupBy is the grouping dimension ("hour", "day", "source", "level").
	GroupBy string `json:"group_by"`
	// Source is the source to filter by (optional).
	Source string `json:"source,omitempty"`
	// Service is the service to filter by (optional).
	Service string `json:"service,omitempty"`
}

// DefaultStatsRequest returns a StatsRequest with defaults.
func DefaultStatsRequest() *StatsRequest {
	now := time.Now()
	return &StatsRequest{
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now,
		GroupBy:   "hour",
	}
}

// isValidLevel checks if a level value is valid.
func isValidLevel(level LogLevel) bool {
	switch level {
	case LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal:
		return true
	case "debug", "info", "warn", "error", "fatal":
		return true
	default:
		return false
	}
}
