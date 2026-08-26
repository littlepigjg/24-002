package model

import (
	"time"
)

// AlertEvent represents an alert triggered by a rule.
type AlertEvent struct {
	// ID is the unique identifier for the alert.
	ID string `json:"id"`
	// RuleID is the ID of the rule that triggered this alert.
	RuleID string `json:"rule_id"`
	// RuleName is the name of the rule that triggered this alert.
	RuleName string `json:"rule_name"`
	// Severity is the severity level of the alert.
	Severity Severity `json:"severity"`
	// Status is the current status of the alert.
	Status AlertStatus `json:"status"`
	// Message is a human-readable description of the alert.
	Message string `json:"message"`
	// Source is the source that triggered the alert.
	Source string `json:"source"`
	// Service is the service associated with the alert.
	Service string `json:"service,omitempty"`
	// Details contains additional information about the alert.
	Details map[string]interface{} `json:"details,omitempty"`
	// TriggeredAt is when the alert was triggered.
	TriggeredAt time.Time `json:"triggered_at"`
	// AcknowledgedAt is when the alert was acknowledged.
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
	// ResolvedAt is when the alert was resolved.
	ResolvedAt *time.Time `json:"resolved_at,omitempty"`
	// AcknowledgedBy is the user who acknowledged the alert.
	AcknowledgedBy string `json:"acknowledged_by,omitempty"`
}

// NewAlertEvent creates a new AlertEvent.
func NewAlertEvent(rule *AlertRule, message string, source string) *AlertEvent {
	now := time.Now()
	return &AlertEvent{
		ID:         GenerateID(),
		RuleID:     rule.ID,
		RuleName:   rule.Name,
		Severity:   rule.Severity,
		Status:     AlertOpen,
		Message:    message,
		Source:     source,
		Details:    make(map[string]interface{}),
		TriggeredAt: now,
	}
}

// Acknowledge marks the alert as acknowledged.
func (a *AlertEvent) Acknowledge(user string) {
	now := time.Now()
	a.Status = AlertAcknowledged
	a.AcknowledgedAt = &now
	a.AcknowledgedBy = user
}

// Resolve marks the alert as resolved.
func (a *AlertEvent) Resolve() {
	now := time.Now()
	a.Status = AlertResolved
	a.ResolvedAt = &now
}

// Ignore marks the alert as ignored.
func (a *AlertEvent) Ignore() {
	now := time.Now()
	a.Status = AlertIgnored
	a.ResolvedAt = &now
}

// IsOpen checks if the alert is still open.
func (a *AlertEvent) IsOpen() bool {
	return a.Status == AlertOpen
}

// ToMap converts the alert to a map for serialization.
func (a *AlertEvent) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"id":        a.ID,
		"rule_id":   a.RuleID,
		"rule_name": a.RuleName,
		"severity":  a.Severity,
		"status":    a.Status,
		"message":   a.Message,
		"source":    a.Source,
		"service":   a.Service,
	}
	detailCopy := make(map[string]interface{}, len(a.Details))
	for k, v := range a.Details {
		detailCopy[k] = v
	}
	result["details"] = detailCopy
	if !a.TriggeredAt.IsZero() {
		result["triggered_at"] = a.TriggeredAt
	}
	if a.AcknowledgedAt != nil {
		result["acknowledged_at"] = *a.AcknowledgedAt
	}
	if a.ResolvedAt != nil {
		result["resolved_at"] = *a.ResolvedAt
	}
	if a.AcknowledgedBy != "" {
		result["acknowledged_by"] = a.AcknowledgedBy
	}
	return result
}

// ToMapWithGuard converts the alert to a map with a guard filter for details.
func (a *AlertEvent) ToMapWithGuard(guard func(key string) bool) map[string]interface{} {
	result := map[string]interface{}{
		"id":        a.ID,
		"rule_id":   a.RuleID,
		"rule_name": a.RuleName,
		"severity":  a.Severity,
		"status":    a.Status,
		"message":   a.Message,
		"source":    a.Source,
		"service":   a.Service,
	}
	detailCopy := make(map[string]interface{})
	for k, v := range a.Details {
		if guard == nil || guard(k) {
			detailCopy[k] = v
		}
	}
	result["details"] = detailCopy
	if !a.TriggeredAt.IsZero() {
		result["triggered_at"] = a.TriggeredAt
	}
	if a.AcknowledgedAt != nil {
		result["acknowledged_at"] = *a.AcknowledgedAt
	}
	if a.ResolvedAt != nil {
		result["resolved_at"] = *a.ResolvedAt
	}
	if a.AcknowledgedBy != "" {
		result["acknowledged_by"] = a.AcknowledgedBy
	}
	return result
}

// AlertFilter defines criteria for filtering alert events.
type AlertFilter struct {
	// Statuses to include.
	Statuses []AlertStatus `json:"statuses,omitempty"`
	// Severities to include.
	Severities []Severity `json:"severities,omitempty"`
	// RuleIDs to include.
	RuleIDs []string `json:"rule_ids,omitempty"`
	// Source to filter on.
	Source string `json:"source,omitempty"`
	// Service to filter on.
	Service string `json:"service,omitempty"`
	// StartTime is the earliest trigger time.
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime is the latest trigger time.
	EndTime *time.Time `json:"end_time,omitempty"`
}

// Matches checks if an alert event matches the filter.
func (f *AlertFilter) Matches(alert *AlertEvent) bool {
	if len(f.Statuses) > 0 {
		found := false
		for _, s := range f.Statuses {
			if s == alert.Status {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.Severities) > 0 {
		found := false
		for _, s := range f.Severities {
			if s == alert.Severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if len(f.RuleIDs) > 0 {
		found := false
		for _, id := range f.RuleIDs {
			if id == alert.RuleID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if f.Source != "" && f.Source != alert.Source {
		return false
	}

	if f.Service != "" && f.Service != alert.Service {
		return false
	}

	if f.StartTime != nil && alert.TriggeredAt.Before(*f.StartTime) {
		return false
	}
	if f.EndTime != nil && alert.TriggeredAt.After(*f.EndTime) {
		return false
	}

	return true
}
