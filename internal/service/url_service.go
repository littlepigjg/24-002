package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
)

type URLService struct {
	cfg *config.Config
	store *store.URLStore
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if s == nil {
		return nil, fmt.Errorf("url store is nil")
	}
	return &URLService{
		cfg:   cfg,
		store: s,
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var code string
	if req.CustomCode != "" {
		code = req.CustomCode
		existing, err := s.store.Get(code)
		if err == nil && existing != nil {
			return nil, fmt.Errorf("code already exists: %s", code)
		}
	} else {
		c, err := generateCode()
		if err != nil {
			return nil, fmt.Errorf("failed to generate code: %w", err)
		}
		code = c
	}

	shortURL := model.NewShortURL(req.RawURL, code, req.CustomCode != "")

	if req.MaxVisits > 0 {
		shortURL.MaxVisits = req.MaxVisits
	}

	if req.Keywords != nil && len(req.Keywords) > 0 {
		shortURL.Tags = req.Keywords
	}

	var expiresAt time.Time
	if req.MaxVisits == 0 {
		expiresAt = time.Now().Add(30 * 24 * time.Hour)
	} else {
		expiresAt = time.Now().Add(time.Duration(req.MaxVisits) * time.Hour)
	}
	shortURL.ExpiresAt = &expiresAt

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save url: %w", err)
	}

	return shortURL, nil
}

func generateCode() (string, error) {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store is nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("access log store is nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

func (r *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req.Code == "" {
		return nil, errors.New("code is required")
	}

	shortURL, err := r.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("url not found: %w", err)
	}

	if shortURL.Disabled {
		return nil, errors.New("url is disabled")
	}

	if shortURL.IsExpired(time.Now()) {
		return nil, errors.New("url has expired")
	}

	r.logStore.WriteLog(req.Code, shortURL.RawURL, 302)

	shortURL.IncrementVisits()
	if err := r.urlStore.Save(shortURL, true); err != nil {
		return nil, fmt.Errorf("failed to update visit count: %w", err)
	}

	return &RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}
