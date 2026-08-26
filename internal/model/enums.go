package model

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"
)

// GenerateID generates a unique ID using crypto/rand.
func GenerateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		// Fallback to a simpler ID if crypto/rand fails
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// GenerateShortID generates a shorter unique ID.
func GenerateShortID() string {
	b := make([]byte, 8)
	_, err := rand.Read(b)
	if err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// RuleStatus represents the status of an alert rule.
type RuleStatus string

const (
	// RuleActive means the rule is actively evaluated.
	RuleActive RuleStatus = "active"
	// RulePaused means the rule is temporarily disabled.
	RulePaused RuleStatus = "paused"
	// RuleArchived means the rule is no longer used.
	RuleArchived RuleStatus = "archived"
)

// RuleConditionType defines the type of condition for a rule.
type RuleConditionType string

const (
	// ConditionCount checks if the count of matching logs exceeds a threshold.
	ConditionCount RuleConditionType = "count"
	// ConditionErrorRate checks if the error rate exceeds a threshold.
	ConditionErrorRate RuleConditionType = "error_rate"
	// ConditionKeyword checks if specific keywords appear in logs.
	ConditionKeyword RuleConditionType = "keyword"
	// ConditionLevel checks if a specific log level appears.
	ConditionLevel RuleConditionType = "level"
)

// AlertStatus represents the status of an alert.
type AlertStatus string

const (
	// AlertOpen means the alert is still active.
	AlertOpen AlertStatus = "open"
	// AlertAcknowledged means the alert has been acknowledged.
	AlertAcknowledged AlertStatus = "acknowledged"
	// AlertResolved means the alert condition is resolved.
	AlertResolved AlertStatus = "resolved"
	// AlertIgnored means the alert was ignored.
	AlertIgnored AlertStatus = "ignored"
)

// Severity represents the severity level of an alert.
type Severity string

const (
	// SeverityLow for low severity alerts.
	SeverityLow Severity = "low"
	// SeverityMedium for medium severity alerts.
	SeverityMedium Severity = "medium"
	// SeverityHigh for high severity alerts.
	SeverityHigh Severity = "high"
	// SeverityCritical for critical severity alerts.
	SeverityCritical Severity = "critical"
)

// AllSeverities returns all severity levels.
func AllSeverities() []Severity {
	return []Severity{SeverityLow, SeverityMedium, SeverityHigh, SeverityCritical}
}

// AllAlertStatuses returns all alert statuses.
func AllAlertStatuses() []AlertStatus {
	return []AlertStatus{AlertOpen, AlertAcknowledged, AlertResolved, AlertIgnored}
}

// AllRuleStatuses returns all rule statuses.
func AllRuleStatuses() []RuleStatus {
	return []RuleStatus{RuleActive, RulePaused, RuleArchived}
}
