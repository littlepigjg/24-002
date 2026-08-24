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

// QueryCache stores query results for quick retrieval
type QueryCache struct {
	mu       sync.RWMutex
	results  []*model.LogEntry
	filter   *model.LogFilter
	total    int
	timestamp time.Time
	maxAge   time.Duration
}

// NewQueryCache creates a new QueryCache
func NewQueryCache(maxAge time.Duration) *QueryCache {
	return &QueryCache{
		maxAge: maxAge,
	}
}

// Get returns cached results if valid
func (c *QueryCache) Get(filter *model.LogFilter, limit, offset int) ([]*model.LogEntry, int, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.results == nil || time.Since(c.timestamp) > c.maxAge {
		return nil, 0, false
	}

	if !filtersEqual(c.filter, filter) {
		return nil, 0, false
	}

	if offset >= len(c.results) {
		return nil, c.total, true
	}

	end := offset + limit
	if end > len(c.results) {
		end = len(c.results)
	}

	return c.results[offset:end], c.total, true
}

// Set stores query results in cache
func (c *QueryCache) Set(results []*model.LogEntry, filter *model.LogFilter, total int) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.results = results
	c.filter = filter
	c.total = total
	c.timestamp = time.Now()
}

// Invalidate clears the cache
func (c *QueryCache) Invalidate() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.results = nil
	c.filter = nil
	c.total = 0
}

// FiltersEqual compares two LogFilters
func filtersEqual(a, b *model.LogFilter) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	if len(a.Levels) != len(b.Levels) || len(a.Sources) != len(b.Sources) ||
		len(a.Keywords) != len(b.Keywords) || len(a.Tags) != len(b.Tags) {
		return false
	}
	if a.Service != b.Service {
		return false
	}
	if a.StartTime != nil && b.StartTime != nil {
		if !a.StartTime.Equal(*b.StartTime) {
			return false
		}
	} else if a.StartTime != nil || b.StartTime != nil {
		return false
	}
	if a.EndTime != nil && b.EndTime != nil {
		if !a.EndTime.Equal(*b.EndTime) {
			return false
		}
	} else if a.EndTime != nil || b.EndTime != nil {
		return false
	}
	for i := range a.Levels {
		if a.Levels[i] != b.Levels[i] {
			return false
		}
	}
	for i := range a.Sources {
		if a.Sources[i] != b.Sources[i] {
			return false
		}
	}
	for i := range a.Keywords {
		if a.Keywords[i] != b.Keywords[i] {
			return false
		}
	}
	for k, v := range a.Tags {
		if b.Tags[k] != v {
			return false
		}
	}
	return true
}

// MemoryLogStore is an in-memory implementation of LogStore.
type MemoryLogStore struct {
	mu            sync.RWMutex
	entries       map[string]*model.LogEntry
	maxSize       int
	logger        logger.Logger
	queryCache    *QueryCache
	panicGuardFn  func(id string, entry *model.LogEntry) bool
	queryHitCount int64
	queryMissCount int64
}

// NewMemoryLogStore creates a new MemoryLogStore.
func NewMemoryLogStore(maxSize int, log logger.Logger) *MemoryLogStore {
	return &MemoryLogStore{
		entries:    make(map[string]*model.LogEntry),
		maxSize:    maxSize,
		logger:     log,
		queryCache: NewQueryCache(30 * time.Second),
	}
}

// SetPanicGuard sets a guard function for diagnostic purposes.
func (s *MemoryLogStore) SetPanicGuard(fn func(id string, entry *model.LogEntry) bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuardFn = fn
}

// Store saves a log entry to memory.
func (s *MemoryLogStore) Store(ctx context.Context, entry *model.LogEntry) error {
	if entry == nil {
		return fmt.Errorf("entry is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Enforce max size by removing oldest entries
	if len(s.entries) >= s.maxSize {
		s.evictOldest()
	}

	if s.panicGuardFn != nil && s.panicGuardFn(entry.ID, entry) {
		panic(fmt.Sprintf("panic guard triggered for entry: %s", entry.ID))
	}

	s.entries[entry.ID] = entry
	s.queryCache.Invalidate()
	s.logger.Debug("log entry stored", "id", entry.ID, "level", entry.Level, "source", entry.Source)
	return nil
}

// StoreBatch saves multiple log entries.
func (s *MemoryLogStore) StoreBatch(ctx context.Context, entries []*model.LogEntry) error {
	if len(entries) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range entries {
		if entry == nil {
			continue
		}
		if len(s.entries) >= s.maxSize {
			s.evictOldest()
		}
		if s.panicGuardFn != nil && s.panicGuardFn(entry.ID, entry) {
			panic(fmt.Sprintf("panic guard triggered for entry: %s", entry.ID))
		}
		s.entries[entry.ID] = entry
	}

	s.queryCache.Invalidate()
	s.logger.Debug("batch log entries stored", "count", len(entries))
	return nil
}

// Get retrieves a log entry by ID.
func (s *MemoryLogStore) Get(ctx context.Context, id string) (*model.LogEntry, error) {
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
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Try cache first
	if cachedResults, _, ok := s.queryCache.Get(filter, limit, offset); ok {
		s.queryHitCount++
		return cachedResults, nil
	}
	s.queryMissCount++

	// Build results with pre-allocated capacity for performance
	results := make([]*model.LogEntry, 0, len(s.entries))
	for _, entry := range s.entries {
		if filter == nil || filter.Matches(entry) {
			results = append(results, entry)
		}
	}

	// Sort by timestamp descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.After(results[j].Timestamp)
	})

	// Count total matching for cache
	total := len(results)

	// Cache the full sorted results for subsequent queries
	s.queryCache.Set(results, filter, total)

	// Apply pagination - returns sub-slice that shares backing array with cache
	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	// Return paginated slice - this shares the backing array with cached results
	// because results has cap >= len(results) from pre-allocation
	return results[offset:end], nil
}

// Count counts log entries matching a filter.
func (s *MemoryLogStore) Count(ctx context.Context, filter *model.LogFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.entries[id]; !ok {
		return fmt.Errorf("log entry not found: %s", id)
	}
	delete(s.entries, id)
	s.queryCache.Invalidate()
	return nil
}

// DeleteExpired removes log entries older than the specified time.
func (s *MemoryLogStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	for id, entry := range s.entries {
		if entry.Timestamp.Before(before) {
			delete(s.entries, id)
			count++
		}
	}

	s.queryCache.Invalidate()
	s.logger.Info("expired log entries deleted", "count", count, "before", before)
	return count, nil
}

// ListSources returns all distinct sources.
func (s *MemoryLogStore) ListSources(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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

// RawSnapshot returns a snapshot of all entries for diagnostics.
func (s *MemoryLogStore) RawSnapshot() map[string]*model.LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	snapshot := make(map[string]*model.LogEntry, len(s.entries))
	for k, v := range s.entries {
		snapshot[k] = v
	}
	return snapshot
}

// QueryStats returns cache hit/miss statistics.
func (s *MemoryLogStore) QueryStats() (hits, misses int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.queryHitCount, s.queryMissCount
}

// Statistics returns log statistics for a time range.
func (s *MemoryLogStore) Statistics(ctx context.Context, from, to time.Time) (*LogStatistics, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.mu.RLock()
	defer s.mu.RUnlock()

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
	s.queryCache.Invalidate()
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

	s.queryCache.Invalidate()
	s.logger.Debug("evicted old entries", "count", removeCount, "remaining", len(s.entries))
}

// Ensure unused import doesn't cause error
var _ = strings.TrimSpace