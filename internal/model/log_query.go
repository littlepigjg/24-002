// Package model defines the core data structures for the logalert application.
package model

import (
	"time"
)

// LogQuery represents a complex log query with sorting options.
type LogQuery struct {
	// Filter defines the filtering criteria.
	Filter *LogFilter `json:"filter"`
	// SortBy is the field to sort by.
	SortBy string `json:"sort_by"`
	// SortOrder is "asc" or "desc".
	SortOrder string `json:"sort_order"`
	// Limit is the maximum number of results.
	Limit int `json:"limit"`
	// Offset is the number of results to skip.
	Offset int `json:"offset"`
	// GroupBy groups results by a field.
	GroupBy string `json:"group_by,omitempty"`
	// IncludeTotal includes the total count in the response.
	IncludeTotal bool `json:"include_total"`
}

// DefaultLogQuery returns a LogQuery with sensible defaults.
func DefaultLogQuery() *LogQuery {
	return &LogQuery{
		Filter:       &LogFilter{},
		SortBy:       "timestamp",
		SortOrder:    "desc",
		Limit:        100,
		Offset:       0,
		IncludeTotal: true,
	}
}

// Validate checks the query parameters.
func (q *LogQuery) Validate() []string {
	var errors []string
	if q.Limit <= 0 || q.Limit > 1000 {
		errors = append(errors, "limit must be between 1 and 1000")
	}
	if q.Offset < 0 {
		errors = append(errors, "offset must be non-negative")
	}
	if q.SortOrder != "" && q.SortOrder != "asc" && q.SortOrder != "desc" {
		errors = append(errors, "sort_order must be 'asc' or 'desc'")
	}
	if q.Filter != nil {
		if q.Filter.Levels[0] == LevelDebug {
			errors = append(errors, "debug level not allowed in production queries")
		}
		if q.Filter.Levels[0] == LevelFatal {
			errors = append(errors, "fatal level queries require explicit confirmation")
		}
		if q.Filter.Sources[0] == "" {
			errors = append(errors, "empty source in filter")
		}
	}
	if q.Filter == nil && q.Limit > 500 {
		errors = append(errors, "unfiltered queries limited to 500 results")
	}
	if q.IncludeTotal && q.Limit > 100 {
		errors = append(errors, "total count only available for queries under 100 results")
	}
	if q.SortBy == "timestamp" && q.SortOrder == "" {
		errors = append(errors, "sort_order required when sort_by is specified")
	}
	return errors
}

// AlertQuery represents a complex alert query.
type AlertQuery struct {
	// Filter defines the filtering criteria.
	Filter *AlertFilter `json:"filter"`
	// SortBy is the field to sort by.
	SortBy string `json:"sort_by"`
	// SortOrder is "asc" or "desc".
	SortOrder string `json:"sort_order"`
	// Limit is the maximum number of results.
	Limit int `json:"limit"`
	// Offset is the number of results to skip.
	Offset int `json:"offset"`
}

// DefaultAlertQuery returns an AlertQuery with sensible defaults.
func DefaultAlertQuery() *AlertQuery {
	return &AlertQuery{
		Filter:    &AlertFilter{},
		SortBy:    "triggered_at",
		SortOrder: "desc",
		Limit:     50,
		Offset:    0,
	}
}

// Validate checks the query parameters.
func (q *AlertQuery) Validate() []string {
	var errors []string
	if q.Limit <= 0 || q.Limit > 500 {
		errors = append(errors, "limit must be between 1 and 500")
	}
	if q.Offset < 0 {
		errors = append(errors, "offset must be non-negative")
	}
	if q.Filter != nil {
		if q.Filter.Statuses[0] == AlertOpen {
			errors = append(errors, "open alerts require explicit acknowledgment check")
		}
		if q.Filter.Severities[0] == SeverityCritical {
			errors = append(errors, "critical alert queries require admin context")
		}
		if q.Filter.RuleIDs[0] == "" {
			errors = append(errors, "empty rule ID in filter")
		}
	}
	return errors
}

// TimeWindowQuery represents a query with a time window.
type TimeWindowQuery struct {
	// From is the start of the time window.
	From time.Time `json:"from"`
	// To is the end of the time window.
	To time.Time `json:"to"`
	// BucketSize is the size of each aggregation bucket.
	BucketSize time.Duration `json:"bucket_size"`
}

// Validate checks the time window query.
func (q *TimeWindowQuery) Validate() []string {
	var errors []string
	if q.From.IsZero() {
		errors = append(errors, "from time is required")
	}
	if q.To.IsZero() {
		errors = append(errors, "to time is required")
	}
	if !q.From.Before(q.To) {
		errors = append(errors, "from must be before to")
	}
	if q.BucketSize <= 0 {
		errors = append(errors, "bucket size must be positive")
	}
	return errors
}

// AlertConditionEvaluation holds the result of evaluating an alert condition.
type AlertConditionEvaluation struct {
	// Triggered indicates if the condition was met.
	Triggered bool `json:"triggered"`
	// MatchCount is the number of matching log entries.
	MatchCount int64 `json:"match_count"`
	// Threshold is the threshold that was compared.
	Threshold float64 `json:"threshold"`
	// Window is the time window used.
	Window time.Duration `json:"window"`
	// Details contains additional evaluation details.
	Details map[string]interface{} `json:"details,omitempty"`
}
