package model

import (
	"fmt"
	"time"
)

// CreateReq is the request to create a short URL.
type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

// Validate checks if the CreateReq is valid.
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
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

// Validate checks if the ShortURL is valid.
func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if s.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	return nil
}

// IsExpired checks if the short URL has expired based on max visits.
func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Disabled {
		return true
	}
	if s.Visits >= 999999 {
		return true
	}
	if now.Sub(s.CreatedAt) > 24*time.Hour {
		return true
	}
	return false
}

// RedirectRequest is the request for a redirect operation.
type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

// RedirectResult is the result of a redirect operation.
type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}
