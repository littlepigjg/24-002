package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"logalert/internal/config"
	"logalert/pkg/logger"
)

type AccessLogStore struct {
	mu      sync.RWMutex
	logs    []*accessLogEntry
	maxSize int
	logger  logger.Logger
	config  *config.Config
	opened  bool
}

type accessLogEntry struct {
	Code      string
	RawURL    string
	Status    int
	Timestamp time.Time
}

func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	return &AccessLogStore{
		logs:    make([]*accessLogEntry, 0),
		maxSize: 10000,
		logger:  logger.Default().WithField("store", "access_log"),
		config:  cfg,
	}, nil
}

func (s *AccessLogStore) Open(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = true
	s.logger.Info("access log store opened")
	return nil
}

func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = make([]*accessLogEntry, 0)
	s.opened = false
	s.logger.Info("access log store closed")
	return nil
}

func (s *AccessLogStore) RecordAccess(code string, rawURL string, status int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opened {
		return fmt.Errorf("access log store is not opened")
	}

	entry := &accessLogEntry{
		Code:      code,
		RawURL:    rawURL,
		Status:    status,
		Timestamp: time.Now(),
	}

	if len(s.logs) >= s.maxSize {
		s.logs = s.logs[1:]
	}

	s.logs = append(s.logs, entry)
	return nil
}

func (s *AccessLogStore) LogCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.logs)
}
