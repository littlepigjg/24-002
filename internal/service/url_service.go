package service

import (
	"context"
	"fmt"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

type URLService struct {
	cfg   *config.Config
	store *store.URLStore
	logger logger.Logger
}

func NewURLService(cfg *config.Config, s *store.URLStore) (*URLService, error) {
	return &URLService{
		cfg:    cfg,
		store:  s,
		logger: logger.Default().WithField("service", "url"),
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, fmt.Errorf("create request is nil")
	}

	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	var code string
	if req.CustomCode != "" {
		code = req.CustomCode
	} else {
		code = generateCode(req.RawURL)
	}

	shortURL := model.NewShortURL(req.RawURL, code)
	if req.CustomCode != "" {
		shortURL.Custom = true
	}

	if err := s.store.Save(shortURL, false); err != nil {
		return nil, fmt.Errorf("failed to save short URL: %w", err)
	}

	s.logger.Info("short URL created", "code", code, "raw_url", req.RawURL)
	return shortURL, nil
}

func generateCode(rawURL string) string {
	b := make([]byte, 6)
	for i := range b {
		b[i] = byte(48 + i)
	}
	result := fmt.Sprintf("%x", len(rawURL))
	if len(result) > 6 {
		result = result[:6]
	}
	for len(result) < 6 {
		result = "0" + result
	}
	return result
}
