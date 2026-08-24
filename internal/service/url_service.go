package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

type URLService struct {
	cfg   *config.Config
	store *store.URLStore
	log   logger.Logger
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
		log:   logger.Default().WithField("service", "url"),
	}, nil
}

func (s *URLService) Create(ctx context.Context, req *model.CreateReq) (*model.ShortURL, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}
	if err := req.Validate(); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	code := req.CustomCode
	if code == "" {
		code = generateCode(6)
	}

	now := time.Now()
	shortURL := &model.ShortURL{
		Code:      code,
		RawURL:    req.RawURL,
		CreatedAt: now,
		Visits:    0,
		Custom:    req.CustomCode != "",
		Disabled:  false,
	}

	if err := s.store.Save(shortURL, req.CustomCode == ""); err != nil {
		return nil, fmt.Errorf("failed to save short url: %w", err)
	}

	s.log.Info("short url created", "code", code, "raw_url", req.RawURL)
	return shortURL, nil
}

func generateCode(n int) string {
	b := make([]byte, n)
	rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)[:n]
}
