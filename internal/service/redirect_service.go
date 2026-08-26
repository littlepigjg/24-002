package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// RedirectService handles URL redirect operations.
type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
	logger    logger.Logger
}

// NewRedirectService creates a new RedirectService.
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("url store is nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("log store is nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logger: logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter()),
	}, nil
}

// HandleRedirect handles a redirect request.
func (s *RedirectService) HandleRedirect(ctx context.Context, req *model.RedirectRequest) (*model.RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context cancelled: %w", err)
	}

	shortURL, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("short URL not found: %w", err)
	}

	if shortURL.IsExpired(time.Now()) {
		return &model.RedirectResult{
			RawURL: shortURL.RawURL,
			Status: 410,
		}, nil
	}

	entry := store.AccessLogEntry{
		Code:      req.Code,
		RawURL:    shortURL.RawURL,
		Timestamp: req.Timestamp,
	}
	if err := s.logStore.Write(entry); err != nil {
		s.logger.Debug("failed to write access log", "error", err)
	}

	return &model.RedirectResult{
		RawURL: shortURL.RawURL,
		Status: 302,
	}, nil
}
