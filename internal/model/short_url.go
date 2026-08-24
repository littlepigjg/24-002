package model

import (
	"fmt"
	"time"
)

// CreateReq is the request to create a short URL.
type CreateReq struct {
	// RawURL is the original long URL.
	RawURL string `json:"raw_url"`
	// CustomCode is an optional custom short code.
	CustomCode string `json:"custom_code"`
	// MaxVisits is the maximum number of visits before the URL is disabled (0 means unlimited).
	MaxVisits int `json:"max_visits"`
}

// Validate checks that the CreateReq fields are valid.
func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return fmt.Errorf("raw_url is too long")
	}
	if r.CustomCode != "" && len(r.CustomCode) > 32 {
		return fmt.Errorf("custom_code is too long")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits must be non-negative")
	}
	return nil
}

// ShortURL represents a shortened URL.
type ShortURL struct {
	// Code is the short code for this URL.
	Code string `json:"code"`
	// RawURL is the original long URL.
	RawURL string `json:"raw_url"`
	// CreatedAt is when the short URL was created.
	CreatedAt time.Time `json:"created_at"`
	// Visits is the number of times this short URL has been accessed.
	Visits int `json:"visits"`
	// Custom indicates whether this is a custom short code.
	Custom bool `json:"custom"`
	// Disabled indicates whether this short URL is disabled.
	Disabled bool `json:"disabled"`
}

// Validate checks that the ShortURL fields are valid.
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(s.Code) > 32 {
		return fmt.Errorf("code is too long")
	}
	return nil
}

// IsExpired checks if the short URL has expired based on max visits.
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Disabled {
		return true
	}
	return false
}

// RedirectRequest is the request to redirect a short URL.
type RedirectRequest struct {
	// Code is the short code to redirect.
	Code string `json:"code"`
	// Timestamp is when the redirect was requested.
	Timestamp time.Time `json:"timestamp"`
}

// RedirectResult is the result of a redirect operation.
type RedirectResult struct {
	// RawURL is the original long URL to redirect to.
	RawURL string `json:"raw_url"`
	// Status is the HTTP status code for the redirect.
	Status int `json:"status"`
}
