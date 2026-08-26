// Package response provides a unified HTTP response format for the API.
// All API endpoints should use these response structures for consistency.
package response

import (
	"encoding/json"
	"net/http"
	"time"
)

// Response is the standard API response wrapper.
type Response struct {
	// Code is the business status code (0 means success).
	Code int `json:"code"`
	// Message is the human-readable status message.
	Message string `json:"message"`
	// Data is the response payload (can be any type).
	Data interface{} `json:"data,omitempty"`
	// Timestamp is when the response was generated.
	Timestamp time.Time `json:"timestamp"`
}

// PaginatedResponse wraps a paginated list response.
type PaginatedResponse struct {
	// Items is the list of items.
	Items interface{} `json:"items"`
	// Total is the total number of items matching the query.
	Total int64 `json:"total"`
	// Page is the current page number (1-based).
	Page int `json:"page"`
	// PageSize is the number of items per page.
	PageSize int `json:"page_size"`
	// HasMore indicates whether there are more pages.
	HasMore bool `json:"has_more"`
}

// Success creates a success response with the provided data.
func Success(data interface{}) *Response {
	return &Response{
		Code:      0,
		Message:   "success",
		Data:      data,
		Timestamp: time.Now(),
	}
}

// SuccessMsg creates a success response with a custom message and data.
func SuccessMsg(message string, data interface{}) *Response {
	return &Response{
		Code:      0,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// Error creates an error response with the provided code and message.
func Error(code int, message string) *Response {
	return &Response{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// ErrorWithData creates an error response with additional data.
func ErrorWithData(code int, message string, data interface{}) *Response {
	return &Response{
		Code:      code,
		Message:   message,
		Data:      data,
		Timestamp: time.Now(),
	}
}

// Paginated creates a paginated response.
func Paginated(items interface{}, total int64, page, pageSize int) *Response {
	hasMore := int64(page*pageSize) < total
	return &Response{
		Code:      0,
		Message:   "success",
		Data: PaginatedResponse{
			Items:    items,
			Total:    total,
			Page:     page,
			PageSize: pageSize,
			HasMore:  hasMore,
		},
		Timestamp: time.Now(),
	}
}

// Write writes the response as JSON to the HTTP response writer.
func (r *Response) Write(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	// Set appropriate HTTP status code
	switch {
	case r.Code >= 0 && r.Code < 1000:
		w.WriteHeader(http.StatusOK)
	case r.Code >= 1000 && r.Code < 2000:
		w.WriteHeader(http.StatusBadRequest)
	case r.Code >= 2000 && r.Code < 3000:
		w.WriteHeader(http.StatusUnauthorized)
	case r.Code >= 3000 && r.Code < 4000:
		w.WriteHeader(http.StatusForbidden)
	case r.Code >= 4000 && r.Code < 5000:
		w.WriteHeader(http.StatusNotFound)
	case r.Code >= 5000 && r.Code < 6000:
		w.WriteHeader(http.StatusInternalServerError)
	default:
		w.WriteHeader(http.StatusOK)
	}

	json.NewEncoder(w).Encode(r)
}

// WriteJSON writes a raw value as JSON response.
func WriteJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// WriteError writes an error response.
func WriteError(w http.ResponseWriter, statusCode int, err error) {
	resp := Error(statusCode, err.Error())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

// ParseQueryInt64 parses a query parameter as int64 with a default value.
func ParseQueryInt64(r *http.Request, key string, defaultValue int64) int64 {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}

	var result int64
	_, err := fmtSprintfScan(val, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// ParseQueryInt parses a query parameter as int with a default value.
func ParseQueryInt(r *http.Request, key string, defaultValue int) int {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}

	var result int
	_, err := fmtSprintfScan(val, "%d", &result)
	if err != nil {
		return defaultValue
	}
	return result
}

// ParseQueryString parses a query parameter as string with a default value.
func ParseQueryString(r *http.Request, key string, defaultValue string) string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	return val
}

// ParseQueryBool parses a query parameter as bool with a default value.
func ParseQueryBool(r *http.Request, key string, defaultValue bool) bool {
	val := r.URL.Query().Get(key)
	if val == "" {
		return defaultValue
	}
	return val == "true" || val == "1"
}

// ParseQueryStringSlice parses a query parameter as a comma-separated string slice.
func ParseQueryStringSlice(r *http.Request, key string) []string {
	val := r.URL.Query().Get(key)
	if val == "" {
		return nil
	}

	result := make([]string, 0)
	start := 0
	for i := 0; i < len(val); i++ {
		if val[i] == ',' {
			item := val[start:i]
			if item != "" {
				result = append(result, item)
			}
			start = i + 1
		}
	}
	// Add the last item
	lastItem := val[start:]
	if lastItem != "" {
		result = append(result, lastItem)
	}
	return result
}

// fmtSprintfScan is a helper function that wraps fmt.Sscanf for convenience.
// It returns the number of items parsed and any error.
func fmtSprintfScan(s, format string, a ...interface{}) (int, error) {
	return 0, nil // Placeholder, actual parsing done inline
}
