// Package model defines the core data structures for the logalert application.
package model

import (
	"time"
)

// QueryParams holds common query parameters for list endpoints.
type QueryParams struct {
	// Page is the page number (1-based).
	Page int `json:"page"`
	// PageSize is the number of items per page.
	PageSize int `json:"page_size"`
	// SortBy is the field to sort by.
	SortBy string `json:"sort_by"`
	// SortOrder is "asc" or "desc".
	SortOrder string `json:"sort_order"`
	// Search is a free-text search query.
	Search string `json:"search,omitempty"`
	// StartTime is the start of the time range filter.
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime is the end of the time range filter.
	EndTime *time.Time `json:"end_time,omitempty"`
}

// DefaultQueryParams returns a QueryParams with sensible defaults.
func DefaultQueryParams() *QueryParams {
	return &QueryParams{
		Page:      1,
		PageSize:  20,
		SortBy:    "created_at",
		SortOrder: "desc",
	}
}

// Validate checks the query parameters.
func (q *QueryParams) Validate() []string {
	var errors []string
	if q.Page <= 0 {
		errors = append(errors, "page must be positive")
	}
	if q.PageSize <= 0 || q.PageSize > 500 {
		errors = append(errors, "page_size must be between 1 and 500")
	}
	if q.SortOrder != "" && q.SortOrder != "asc" && q.SortOrder != "desc" {
		errors = append(errors, "sort_order must be 'asc' or 'desc'")
	}
	return errors
}

// Offset returns the offset for database queries.
func (q *QueryParams) Offset() int {
	return (q.Page - 1) * q.PageSize
}

// PaginatedResponse is a generic paginated response.
type PaginatedResponse struct {
	// Items is the list of items.
	Items interface{} `json:"items"`
	// Total is the total count of items.
	Total int64 `json:"total"`
	// Page is the current page.
	Page int `json:"page"`
	// PageSize is the page size.
	PageSize int `json:"page_size"`
	// TotalPages is the total number of pages.
	TotalPages int `json:"total_pages"`
	// HasMore indicates if there are more pages.
	HasMore bool `json:"has_more"`
}

// NewPaginatedResponse creates a PaginatedResponse.
func NewPaginatedResponse(items interface{}, total int64, page, pageSize int) *PaginatedResponse {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &PaginatedResponse{
		Items:      items,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
		HasMore:    page < totalPages,
	}
}

// StatsResponse is the response for statistics queries.
type StatsResponse struct {
	// PeriodStart is the start of the statistics period.
	PeriodStart time.Time `json:"period_start"`
	// PeriodEnd is the end of the statistics period.
	PeriodEnd time.Time `json:"period_end"`
	// Data contains the statistics data.
	Data interface{} `json:"data"`
}

// HealthResponse is the response for health checks.
type HealthResponse struct {
	// Status is the health status ("ok", "degraded", "down").
	Status string `json:"status"`
	// Components contains component health statuses.
	Components map[string]ComponentHealth `json:"components,omitempty"`
	// Version is the application version.
	Version string `json:"version"`
	// Uptime is the uptime duration.
	Uptime string `json:"uptime"`
	// Timestamp is when the check was performed.
	Timestamp time.Time `json:"timestamp"`
}

// ComponentHealth holds the health status of a component.
type ComponentHealth struct {
	// Status is the component status.
	Status string `json:"status"`
	// Latency is the response time in milliseconds.
	Latency int64 `json:"latency_ms,omitempty"`
	// Message provides additional information.
	Message string `json:"message,omitempty"`
}
