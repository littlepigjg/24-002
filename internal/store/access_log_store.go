package store

import (
	"context"
	"fmt"
	"sync"

	"logalert/internal/config"
	"logalert/pkg/cache"
	"logalert/pkg/logger"
)

type AccessLogStore struct {
	mu      sync.RWMutex
	cfg     *config.Config
	cache   *cache.Cache
	logger  logger.Logger
	opened  bool
	records int
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}

	log := logger.Default()
	return &AccessLogStore{
		cfg:    cfg,
		cache:  cache.New(5000),
		logger: log.WithField("component", "access_log_store"),
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.opened {
		return fmt.Errorf("access log store already opened")
	}
	s.opened = true
	s.logger.Info("access log store opened")
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opened {
		return nil
	}
	s.opened = false
	s.logger.Info("access log store closed")
	return nil
}

func (s *AccessLogStore) RecordAccess(code string, rawURL string) error {
	if !s.opened {
		return fmt.Errorf("access log store not opened")
	}

	s.cache.Set(code, rawURL, 0)
	s.mu.Lock()
	s.records++
	s.mu.Unlock()

	return nil
}

func (s *AccessLogStore) GetAccessLog(code string) (string, bool) {
	val, ok := s.cache.GetString(code)
	return val, ok
}

func (s *AccessLogStore) AccessCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.records
}
