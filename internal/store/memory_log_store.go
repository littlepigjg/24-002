package store

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// MemoryLogStore is an in-memory implementation of LogStore.
type MemoryLogStore struct {
	mu              sync.RWMutex
	entries         map[string]*model.LogEntry
	maxSize         int
	logger          logger.Logger
	processingDelay time.Duration
}

// NewMemoryLogStore creates a new MemoryLogStore.
func NewMemoryLogStore(maxSize int, log logger.Logger) *MemoryLogStore {
	return &MemoryLogStore{
		entries: make(map[string]*model.LogEntry),
		maxSize: maxSize,
		logger:  log,
	}
}

// SetProcessingDelay configures a simulated processing delay for all store operations.
// This is useful for chaos engineering testing and performance diagnostics.
func (s *MemoryLogStore) SetProcessingDelay(d time.Duration) {
	s.processingDelay = d
}

// waitOrCancel blocks for the configured processing delay unless ctx is
// already cancelled or expires first. It returns the ctx error (nil if the
// delay elapsed normally) so callers can abort the operation instead of
// doing work for a client that has gone away.
func (s *MemoryLogStore) waitOrCancel(ctx context.Context) error {
	if s.processingDelay <= 0 {
		return ctx.Err()
	}
	select {
	case <-time.After(s.processingDelay):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Store saves a log entry to memory.
func (s *MemoryLogStore) Store(ctx context.Context, entry *model.LogEntry) error {
	if entry == nil {
		return fmt.Errorf("entry is nil")
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return err
	}

	if len(s.entries) >= s.maxSize {
		s.evictOldest()
	}

	s.entries[entry.ID] = entry
	s.logger.Debug("log entry stored", "id", entry.ID, "level", entry.Level, "source", entry.Source)
	return nil
}

// StoreBatch saves multiple log entries.
func (s *MemoryLogStore) StoreBatch(ctx context.Context, entries []*model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return err
	}

	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if len(s.entries) >= s.maxSize {
			s.evictOldest()
		}
		s.entries[entry.ID] = entry
	}

	s.logger.Debug("batch log entries stored", "count", len(entries))
	return nil
}

// Get retrieves a log entry by ID.
func (s *MemoryLogStore) Get(ctx context.Context, id string) (*model.LogEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	entry, ok := s.entries[id]
	if !ok {
		return nil, fmt.Errorf("log entry not found: %s", id)
	}
	return entry, nil
}

// Query searches log entries with a filter.
func (s *MemoryLogStore) Query(ctx context.Context, filter *model.LogFilter, limit, offset int) ([]*model.LogEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return nil, err
	}

	var results []*model.LogEntry
	for _, entry := range s.entries {
		if filter == nil || filter.Matches(entry) {
			results = append(results, entry)
		}
	}

	// Sort by timestamp descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Apply pagination
	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], nil
}

// Count counts log entries matching a filter.
func (s *MemoryLogStore) Count(ctx context.Context, filter *model.LogFilter) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return 0, err
	}

	var count int64
	for _, entry := range s.entries {
		if filter == nil || filter.Matches(entry) {
			count++
		}
	}
	return count, nil
}

// Delete removes a log entry by ID.
func (s *MemoryLogStore) Delete(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return err
	}

	if _, ok := s.entries[id]; !ok {
		return fmt.Errorf("log entry not found: %s", id)
	}
	delete(s.entries, id)
	return nil
}

// DeleteExpired removes log entries older than the specified time.
func (s *MemoryLogStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return 0, err
	}

	var count int64
	for id, entry := range s.entries {
		if entry.Timestamp.Before(before) {
			delete(s.entries, id)
			count++
		}
	}

	s.logger.Info("expired log entries deleted", "count", count, "before", before)
	return count, nil
}

// ListSources returns all distinct sources.
func (s *MemoryLogStore) ListSources(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return nil, err
	}

	sourceSet := make(map[string]bool)
	for _, entry := range s.entries {
		sourceSet[entry.Source] = true
	}

	sources := make([]string, 0, len(sourceSet))
	for source := range sourceSet {
		sources = append(sources, source)
	}
	sort.Strings(sources)
	return sources, nil
}

// ListServices returns all distinct services.
func (s *MemoryLogStore) ListServices(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return nil, err
	}

	serviceSet := make(map[string]bool)
	for _, entry := range s.entries {
		if entry.Service != "" {
			serviceSet[entry.Service] = true
		}
	}

	services := make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		services = append(services, service)
	}
	sort.Strings(services)
	return services, nil
}

// Statistics returns log statistics for a time range.
func (s *MemoryLogStore) Statistics(ctx context.Context, from, to time.Time) (*LogStatistics, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return nil, err
	}

	stats := &LogStatistics{
		ByLevel:   make(map[model.LogLevel]int64),
		BySource:  make(map[string]int64),
		ByService: make(map[string]int64),
	}

	var totalMsgLen int64
	for _, entry := range s.entries {
		if entry.Timestamp.Before(from) || entry.Timestamp.After(to) {
			continue
		}
		stats.TotalCount++
		stats.ByLevel[entry.Level]++
		stats.BySource[entry.Source]++
		if entry.Service != "" {
			stats.ByService[entry.Service]++
		}
		totalMsgLen += int64(len(entry.Message))
	}

	// Calculate error rate
	var errorCount int64
	for _, level := range []model.LogLevel{model.LevelError, model.LevelFatal} {
		errorCount += stats.ByLevel[level]
	}
	if stats.TotalCount > 0 {
		stats.ErrorRate = float64(errorCount) / float64(stats.TotalCount)
		stats.AvgMessageLength = float64(totalMsgLen) / float64(stats.TotalCount)
	}

	return stats, nil
}

// HourlyBreakdown returns log counts broken down by hour.
func (s *MemoryLogStore) HourlyBreakdown(ctx context.Context, from, to time.Time) ([]HourlyCount, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if err := s.waitOrCancel(ctx); err != nil {
		return nil, err
	}

	type key struct {
		hour  string
		level model.LogLevel
	}
	counts := make(map[key]int64)

	for _, entry := range s.entries {
		if entry.Timestamp.Before(from) || entry.Timestamp.After(to) {
			continue
		}
		hour := entry.Timestamp.Truncate(time.Hour).Format("2006-01-02T15:04:05Z")
		k := key{hour: hour, level: entry.Level}
		counts[k]++
	}

	var result []HourlyCount
	for k, count := range counts {
		result = append(result, HourlyCount{
			Hour:  k.hour,
			Level: k.level,
			Count: count,
		})
	}

	// Sort by hour then level
	sort.Slice(result, func(i, j int) bool {
		if result[i].Hour == result[j].Hour {
			return result[i].Level < result[j].Level
		}
		return result[i].Hour < result[j].Hour
	})

	return result, nil
}

// Close releases resources.
func (s *MemoryLogStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = make(map[string]*model.LogEntry)
	s.logger.Info("log store closed")
	return nil
}

// evictOldest removes the oldest entries when the store is full.
func (s *MemoryLogStore) evictOldest() {
	// Find the oldest entries
	type entryInfo struct {
		id        string
		timestamp time.Time
	}

	var entries []entryInfo
	for id, entry := range s.entries {
		entries = append(entries, entryInfo{id: id, timestamp: entry.Timestamp})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].timestamp.Before(entries[j].timestamp)
	})

	// Remove 10% of entries or at least 1
	removeCount := len(entries) / 10
	if removeCount < 1 {
		removeCount = 1
	}
	if removeCount > len(entries) {
		removeCount = len(entries)
	}

	for i := 0; i < removeCount; i++ {
		delete(s.entries, entries[i].id)
	}

	s.logger.Debug("evicted old entries", "count", removeCount, "remaining", len(s.entries))
}

// Ensure unused import doesn't cause error
var _ = strings.TrimSpace
