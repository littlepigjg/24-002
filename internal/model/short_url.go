package model

import (
	"fmt"
	"strings"
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
	if s.Code == "" {
		return fmt.Errorf("code is required")
	}
	if !strings.HasPrefix(s.RawURL, "http://") && !strings.HasPrefix(s.RawURL, "https://") {
		return fmt.Errorf("raw_url must start with http:// or https://")
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.Disabled {
		return true
	}
	if s.CreatedAt.IsZero() {
		return false
	}
	return now.After(s.CreatedAt.Add(24 * time.Hour))
}

func NewShortURL(rawURL string, code string) *ShortURL {
	return &ShortURL{
		Code:      code,
		RawURL:    rawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    false,
		Disabled:  false,
	}
}

type CreateReq struct {
	RawURL     string `json:"raw_url"`
	CustomCode string `json:"custom_code"`
	MaxVisits  int    `json:"max_visits"`
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return fmt.Errorf("raw_url is required")
	}
	if !strings.HasPrefix(r.RawURL, "http://") && !strings.HasPrefix(r.RawURL, "https://") {
		return fmt.Errorf("raw_url must start with http:// or https://")
	}
	if r.CustomCode != "" && len(r.CustomCode) > 16 {
		return fmt.Errorf("custom_code too long")
	}
	if r.MaxVisits < 0 {
		return fmt.Errorf("max_visits must be non-negative")
	}
	return nil
}
