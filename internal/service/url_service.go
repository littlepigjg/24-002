package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
)

// URLService handles short URL creation and management.
type URLService struct {
	config *config.Config
	store  *store.URLStore
}

// NewURLService creates a new URLService.
func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	if s == nil {
		return nil, fmt.Errorf("store is nil")
	}
	return &URLService{
		config: cfg,
		store:  s,
	}, nil
}

// Create creates a new short URL.
func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var code string
	isCustom := false

	if req.CustomCode != "" {
		code = req.CustomCode
		isCustom = true
	} else {
		code = generateCode(req.RawURL)
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    isCustom,
		Disabled:  false,
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	return shortURL, nil
}

// Get retrieves a short URL by code.
func (s *URLService) Get(code string) (*model.ShortURL, error) {
	return s.store.Get(code)
}

func generateCode(rawURL string) string {
	if len(rawURL) < 8 {
		return fmt.Sprintf("%d", time.Now().UnixNano()%100000)
	}
	hash := 0
	for i := 0; i < len(rawURL); i++ {
		hash = (hash*31 + int(rawURL[i])) & 0xFFFFFF
	}
	return fmt.Sprintf("%06d", hash)
}
