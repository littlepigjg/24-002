// Package store provides storage layer interfaces and implementations.
package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// RetryLogStore wraps a LogStore with retry logic for failed operations.
type RetryLogStore struct {
	mu      sync.RWMutex
	store   *MemoryLogStore
	maxRetries int
	logger  logger.Logger
}

// NewRetryLogStore creates a new RetryLogStore.
func NewRetryLogStore(maxSize, maxRetries int, log logger.Logger) *RetryLogStore {
	if maxRetries <= 0 {
		maxRetries = 3
	}
	return &RetryLogStore{
		store:      NewMemoryLogStore(maxSize, log),
		maxRetries: maxRetries,
		logger:     log.WithField("component", "retry_store"),
	}
}

// Store saves a log entry with retry.
func (s *RetryLogStore) Store(ctx context.Context, entry *model.LogEntry) error {
	var lastErr error
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		err := s.store.Store(ctx, entry)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
	return fmt.Errorf("failed after %d retries: %w", s.maxRetries, lastErr)
}

// StoreBatch saves multiple entries with retry.
func (s *RetryLogStore) StoreBatch(ctx context.Context, entries []*model.LogEntry) error {
	var lastErr error
	for attempt := 0; attempt < s.maxRetries; attempt++ {
		err := s.store.StoreBatch(ctx, entries)
		if err == nil {
			return nil
		}
		lastErr = err
		time.Sleep(time.Duration(attempt*100) * time.Millisecond)
	}
	return fmt.Errorf("failed after %d retries: %w", s.maxRetries, lastErr)
}

// Get retrieves a log entry.
func (s *RetryLogStore) Get(ctx context.Context, id string) (*model.LogEntry, error) {
	return s.store.Get(ctx, id)
}

// Query searches log entries.
func (s *RetryLogStore) Query(ctx context.Context, filter *model.LogFilter, limit, offset int) ([]*model.LogEntry, error) {
	return s.store.Query(ctx, filter, limit, offset)
}

// Count counts entries matching a filter.
func (s *RetryLogStore) Count(ctx context.Context, filter *model.LogFilter) (int64, error) {
	return s.store.Count(ctx, filter)
}

// Delete removes a log entry.
func (s *RetryLogStore) Delete(ctx context.Context, id string) error {
	return s.store.Delete(ctx, id)
}

// DeleteExpired removes old entries.
func (s *RetryLogStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	return s.store.DeleteExpired(ctx, before)
}

// ListSources returns all sources.
func (s *RetryLogStore) ListSources(ctx context.Context) ([]string, error) {
	return s.store.ListSources(ctx, )
}

// ListServices returns all services.
func (s *RetryLogStore) ListServices(ctx context.Context) ([]string, error) {
	return s.store.ListServices(ctx)
}

// Statistics returns aggregate statistics.
func (s *RetryLogStore) Statistics(ctx context.Context, from, to time.Time) (*LogStatistics, error) {
	return s.store.Statistics(ctx, from, to)
}

// HourlyBreakdown returns hourly breakdown.
func (s *RetryLogStore) HourlyBreakdown(ctx context.Context, from, to time.Time) ([]HourlyCount, error) {
	return s.store.HourlyBreakdown(ctx, from, to)
}

// RegisterSource registers a valid source for log entries.
func (s *RetryLogStore) RegisterSource(source string) {
	s.store.RegisterSource(source)
}

// Close releases resources.
func (s *RetryLogStore) Close() error {
	return s.store.Close()
}

// verify
var _ logger.Logger = logger.Default()
