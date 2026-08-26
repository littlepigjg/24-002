// Package store provides storage layer interfaces and implementations.
package store

import (
	"context"
	"time"

	"logalert/internal/model"
)

// LogStore defines the interface for log entry storage.
type LogStore interface {
	// Store saves a log entry.
	Store(ctx context.Context, entry *model.LogEntry) error
	// StoreBatch saves multiple log entries.
	StoreBatch(ctx context.Context, entries []*model.LogEntry) error
	// Get retrieves a log entry by ID.
	Get(ctx context.Context, id string) (*model.LogEntry, error)
	// Query searches log entries with a filter.
	Query(ctx context.Context, filter *model.LogFilter, limit, offset int) ([]*model.LogEntry, error)
	// Count counts log entries matching a filter.
	Count(ctx context.Context, filter *model.LogFilter) (int64, error)
	// Delete removes a log entry by ID.
	Delete(ctx context.Context, id string) error
	// DeleteExpired removes log entries older than the specified time.
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
	// ListSources returns all distinct sources.
	ListSources(ctx context.Context) ([]string, error)
	// ListServices returns all distinct services.
	ListServices(ctx context.Context) ([]string, error)
	// Statistics returns log statistics for a time range.
	Statistics(ctx context.Context, from, to time.Time) (*LogStatistics, error)
	// HourlyBreakdown returns log counts broken down by hour.
	HourlyBreakdown(ctx context.Context, from, to time.Time) ([]HourlyCount, error)
	// Close releases resources held by the store.
	Close() error
}

// LogStatistics holds aggregate log statistics.
type LogStatistics struct {
	// TotalCount is the total number of log entries.
	TotalCount int64 `json:"total_count"`
	// ByLevel contains counts grouped by level.
	ByLevel map[model.LogLevel]int64 `json:"by_level"`
	// BySource contains counts grouped by source.
	BySource map[string]int64 `json:"by_source"`
	// ByService contains counts grouped by service.
	ByService map[string]int64 `json:"by_service"`
	// ErrorRate is the ratio of error logs to total logs.
	ErrorRate float64 `json:"error_rate"`
	// AvgMessageLength is the average message length.
	AvgMessageLength float64 `json:"avg_message_length"`
}

// HourlyCount holds log count for a specific hour.
type HourlyCount struct {
	// Hour is the hour (in RFC3339 format).
	Hour string `json:"hour"`
	// Level is the log level.
	Level model.LogLevel `json:"level"`
	// Count is the number of logs.
	Count int64 `json:"count"`
}

// RuleStore defines the interface for alert rule storage.
type RuleStore interface {
	// Create saves a new alert rule.
	Create(ctx context.Context, rule *model.AlertRule) error
	// Get retrieves a rule by ID.
	Get(ctx context.Context, id string) (*model.AlertRule, error)
	// Update updates an existing rule.
	Update(ctx context.Context, rule *model.AlertRule) error
	// Delete removes a rule by ID.
	Delete(ctx context.Context, id string) error
	// List returns all rules.
	List(ctx context.Context) ([]*model.AlertRule, error)
	// ListActive returns all active rules.
	ListActive(ctx context.Context) ([]*model.AlertRule, error)
	// GetBySource returns rules for a specific source.
	GetBySource(ctx context.Context, source string) ([]*model.AlertRule, error)
	// Count returns the total number of rules.
	Count(ctx context.Context) (int64, error)
	// Close releases resources held by the store.
	Close() error
}

// AlertStore defines the interface for alert event storage.
type AlertStore interface {
	// Record saves a new alert event.
	Record(ctx context.Context, alert *model.AlertEvent) error
	// Get retrieves an alert by ID.
	Get(ctx context.Context, id string) (*model.AlertEvent, error)
	// UpdateStatus updates the status of an alert.
	UpdateStatus(ctx context.Context, id string, status model.AlertStatus) error
	// Query searches alerts with a filter.
	Query(ctx context.Context, filter *model.AlertFilter, limit, offset int) ([]*model.AlertEvent, error)
	// Count counts alerts matching a filter.
	Count(ctx context.Context, filter *model.AlertFilter) (int64, error)
	// GetByRule returns alerts for a specific rule.
	GetByRule(ctx context.Context, ruleID string, limit, offset int) ([]*model.AlertEvent, error)
	// GetByStatus returns alerts with a specific status.
	GetByStatus(ctx context.Context, status model.AlertStatus, limit, offset int) ([]*model.AlertEvent, error)
	// ListRecent returns the most recent alerts.
	ListRecent(ctx context.Context, limit int) ([]*model.AlertEvent, error)
	// Delete removes an alert by ID.
	Delete(ctx context.Context, id string) error
	// DeleteOld removes alerts older than the specified time.
	DeleteOld(ctx context.Context, before time.Time) (int64, error)
	// Close releases resources held by the store.
	Close() error
}

// StoreFactory creates store instances.
type StoreFactory interface {
	// CreateLogStore creates a new LogStore.
	CreateLogStore(ctx context.Context) (LogStore, error)
	// CreateRuleStore creates a new RuleStore.
	CreateRuleStore(ctx context.Context) (RuleStore, error)
	// CreateAlertStore creates a new AlertStore.
	CreateAlertStore(ctx context.Context) (AlertStore, error)
}
