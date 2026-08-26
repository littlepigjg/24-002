package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/store"
	"logalert/pkg/logger"
)

// RedirectRequest represents a redirect request.
type RedirectRequest struct {
	Code      string    `json:"code"`
	Timestamp time.Time `json:"timestamp"`
}

// RedirectResult represents a redirect result.
type RedirectResult struct {
	RawURL string `json:"raw_url"`
	Status int    `json:"status"`
}

type RedirectService struct {
	urlStore  *store.URLStore
	logStore  *store.AccessLogStore
	logger    logger.Logger
}

func NewRedirectService(us *store.URLStore, ls *store.AccessLogStore) (*RedirectService, error) {
	return &RedirectService{
		urlStore: us,
		logStore: ls,
		logger:   logger.Default().WithField("service", "redirect"),
	}, nil
}

func (s *RedirectService) HandleRedirect(ctx context.Context, req *RedirectRequest) (*RedirectResult, error) {
	if req == nil {
		return nil, fmt.Errorf("redirect request is nil")
	}

	entry, err := s.urlStore.Get(req.Code)
	if err != nil {
		s.recordAccess(req.Code, "", 404)
		return &RedirectResult{RawURL: "", Status: 404}, nil
	}

	if entry.Disabled {
		s.recordAccess(req.Code, entry.RawURL, 410)
		return &RedirectResult{RawURL: entry.RawURL, Status: 410}, nil
	}

	if entry.IsExpired(time.Now()) {
		s.recordAccess(req.Code, entry.RawURL, 410)
		return &RedirectResult{RawURL: entry.RawURL, Status: 410}, nil
	}

	if err := s.urlStore.IncrVisits(req.Code); err != nil {
		s.logger.Warn("failed to increment visits", "code", req.Code, "error", err)
	}

	s.recordAccess(req.Code, entry.RawURL, 302)

	s.logger.Info("redirect handled", "code", req.Code, "raw_url", entry.RawURL, "visits", entry.Visits)
	return &RedirectResult{RawURL: entry.RawURL, Status: 302}, nil
}

func (s *RedirectService) recordAccess(code string, rawURL string, status int) {
	if s.logStore != nil {
		_ = s.logStore.RecordAccess(code, rawURL, status)
	}
}
