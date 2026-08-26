package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"logalert/internal/config"
	"logalert/pkg/logger"
)

// AccessLogStore stores access logs.
type AccessLogStore struct {
	mu     sync.RWMutex
	logs   []AccessLogEntry
	cfg    *config.Config
	logger logger.Logger
	opened bool
}

// AccessLogEntry represents an access log entry.
type AccessLogEntry struct {
	Code      string    `json:"code"`
	RawURL    string    `json:"raw_url"`
	Timestamp time.Time `json:"timestamp"`
	UserAgent string    `json:"user_agent"`
	IP        string    `json:"ip"`
	Referer   string    `json:"referer"`
}

// NewAccessLogStore creates a new AccessLogStore.
func NewAccessLogStore(cfg *config.Config) (*AccessLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	return &AccessLogStore{
		logs:   make([]AccessLogEntry, 0),
		cfg:    cfg,
		logger: logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter()),
	}, nil
}

// Open opens the access log store.
func (s *AccessLogStore) Open(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return fmt.Errorf("context cancelled during open: %w", err)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.opened = true
	return nil
}

// Write writes an access log entry.
func (s *AccessLogStore) Write(entry AccessLogEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.opened {
		return fmt.Errorf("access log store not opened")
	}

	s.logs = append(s.logs, entry)
	if len(s.logs) > 10000 {
		s.logs = s.logs[len(s.logs)-5000:]
	}
	return nil
}

// Close releases resources held by the AccessLogStore.
func (s *AccessLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.opened = false
	s.logs = make([]AccessLogEntry, 0)
	return nil
}
