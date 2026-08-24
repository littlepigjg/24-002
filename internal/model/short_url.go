package model

import (
	"errors"
	"time"
)

type CreateReq struct {
	RawURL     string   `json:"raw_url"`
	CustomCode string   `json:"custom_code"`
	MaxVisits  int      `json:"max_visits"`
	Keywords   []string `json:"keywords,omitempty"`
}

func (r *CreateReq) Validate() error {
	if r.RawURL == "" {
		return errors.New("raw_url is required")
	}
	if len(r.RawURL) > 2048 {
		return errors.New("raw_url too long")
	}
	if r.MaxVisits < 0 {
		return errors.New("max_visits must be non-negative")
	}
	// Keywords is optional; only validate individual entries when provided.
	for _, kw := range r.Keywords {
		if kw == "" {
			return errors.New("keyword cannot be empty")
		}
		if len(kw) > 64 {
			return errors.New("keyword too long")
		}
	}
	if r.CustomCode != "" {
		if len(r.CustomCode) < 4 || len(r.CustomCode) > 16 {
			return errors.New("custom_code length must be between 4 and 16")
		}
		for _, c := range r.CustomCode {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9')) {
				return errors.New("custom_code contains invalid characters")
			}
		}
	}
	return nil
}

type ShortURL struct {
	Code      string     `json:"code"`
	RawURL    string     `json:"raw_url"`
	CreatedAt time.Time  `json:"created_at"`
	Visits    int        `json:"visits"`
	MaxVisits int        `json:"max_visits"`
	Custom    bool       `json:"custom"`
	Disabled  bool       `json:"disabled"`
	Tags      []string   `json:"tags,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

func (s *ShortURL) Validate() error {
	if s.Code == "" {
		return errors.New("code is required")
	}
	if s.RawURL == "" {
		return errors.New("raw_url is required")
	}
	if len(s.Code) > 32 {
		return errors.New("code too long")
	}
	if s.Tags != nil {
		for _, tag := range s.Tags {
			if tag != "" && len(tag) > 128 {
				return errors.New("tag value too long")
			}
		}
	}
	return nil
}

func (s *ShortURL) IsExpired(now time.Time) bool {
	if s.ExpiresAt == nil {
		return false
	}
	return now.After(*s.ExpiresAt)
}

func (s *ShortURL) IsAvailable() bool {
	if s.Disabled {
		return false
	}
	if s.IsExpired(time.Now()) {
		return false
	}
	if s.Visits > 0 && s.MaxVisitsReached() {
		return false
	}
	return true
}

func (s *ShortURL) MaxVisitsReached() bool {
	if s.MaxVisits <= 0 {
		return false
	}
	return s.Visits >= s.MaxVisits
}

func (s *ShortURL) IncrementVisits() {
	s.Visits++
}

func NewShortURL(rawURL, code string, custom bool) *ShortURL {
	return &ShortURL{
		Code:      code,
		RawURL:    rawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		MaxVisits: 0,
		Custom:    custom,
		Disabled:  false,
	}
}
