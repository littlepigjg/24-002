package model

import (
	"fmt"
	"time"
)

type ShortURL struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	CreatedAt time.Time `json:"created_at"`
	Visits    int       `json:"visits"`
	Custom    bool      `json:"custom"`
	Disabled  bool      `json:"disabled"`
}

func (s *ShortURL) Validate() error {
	if s.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(s.Code) == 0 {
		return fmt.Errorf("code is required")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.CreatedAt.IsZero() {
		return false
	}
	return now.Sub(s.CreatedAt) > 90*24*time.Hour
}

type CreateReq struct {
	RawURL    string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits int    `json:"max_visits"`
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if len(r.CustomCode) > 8 {
		return fmt.Errorf("custom_code must be at most 8 characters")
	}
	return nil
}


