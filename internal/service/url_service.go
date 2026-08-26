package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// URLService handles short URL operations.
type URLService struct {
	cfg    *config.Config
	store  *store.URLStore
	logger logger.Logger
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
		cfg:    cfg,
		store:  s,
		logger: logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter()),
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

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	code := req.CustomCode
	if code == "" {
		code = generateShortCode(req.RawURL)
	}

	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: time.Now(),
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	return shortURL, nil
}

func generateShortCode(rawURL string) string {
	if len(rawURL) == 0 {
		return ""
	}
	hash := 0
	for i := 0; i < len(rawURL); i++ {
		hash = (hash*31 + int(rawURL[i])) & 0x7FFFFFFF
	}
	return fmt.Sprintf("%x", hash)
}
