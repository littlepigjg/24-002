package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/store"
	"logalert/pkg/logger"
)

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
	log       logger.Logger
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
		log:      logger.Default().WithField("service", "redirect"),
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	url, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get url: %w", err)
	}

	if url.Disabled {
		return nil, fmt.Errorf("url is disabled: %s", req.Code)
	}

	if err := s.logStore.RecordAccess(req.Code, url.RawURL); err != nil {
		s.log.Warn("failed to record access log", "error", err, "code", req.Code)
	}

	return &RedirectResult{
		RawURL: url.RawURL,
		Status: 302,
	}, nil
}
