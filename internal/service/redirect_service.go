package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
)

// RedirectService handles URL redirect operations.
type RedirectService struct {
	urlStore   *store.URLStore
	logStore   *store.AccessLogStore
}

// NewRedirectService creates a new RedirectService.
func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	if us == nil {
		return nil, fmt.Errorf("URL store is nil")
	}
	if ls == nil {
		return nil, fmt.Errorf("log store is nil")
	}
	return &RedirectService{
		urlStore: us,
		logStore: ls,
	}, nil
}

// HandleRedirect processes a redirect request.
func (s *RedirectService) HandleRedirect(ctx context.Context, req *model.RedirectRequest) (*model.RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	u, err := s.urlStore.Get(req.Code)
	if err != nil {
		return nil, fmt.Errorf("failed to get URL: %w", err)
	}

	if u.IsExpired(time.Now()) {
		return nil, fmt.Errorf("short URL is disabled or expired")
	}

	s.logStore.AppendLog(store.AccessLogEntry{
		Code:      req.Code,
		RawURL:    u.RawURL,
		Timestamp: req.Timestamp.Format(time.RFC3339),
	})

	return &model.RedirectResult{
		RawURL: u.RawURL,
		Status: 302,
	}, nil
}
