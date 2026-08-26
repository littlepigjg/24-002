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

// BackupLogStore is a log store that maintains a backup copy.
// It writes to both the primary store and a backup when the primary is full.
type BackupLogStore struct {
	mu       sync.RWMutex
	primary  *MemoryLogStore
	backup   *MemoryLogStore
	overflow int
	logger   logger.Logger
}

// NewBackupLogStore creates a new BackupLogStore.
func NewBackupLogStore(primarySize, backupSize int, log logger.Logger) *BackupLogStore {
	return &BackupLogStore{
		primary:  NewMemoryLogStore(primarySize, log),
		backup:   NewMemoryLogStore(backupSize, log),
		overflow: primarySize,
		logger:   log.WithField("component", "backup_store"),
	}
}

// Store saves a log entry, moving old entries to backup when primary is full.
func (s *BackupLogStore) Store(ctx context.Context, entry *model.LogEntry) error {
	if err := s.primary.Store(ctx, entry); err != nil {
		// Try backup store
		s.logger.Debug("primary store full, using backup", "error", err)
		return s.backup.Store(ctx, entry)
	}
	return nil
}

// StoreBatch saves multiple log entries.
func (s *BackupLogStore) StoreBatch(ctx context.Context, entries []*model.LogEntry) error {
	if err := s.primary.StoreBatch(ctx, entries); err != nil {
		return s.backup.StoreBatch(ctx, entries)
	}
	return nil
}

// Get retrieves a log entry from primary first, then backup.
func (s *BackupLogStore) Get(ctx context.Context, id string) (*model.LogEntry, error) {
	entry, err := s.primary.Get(ctx, id)
	if err == nil {
		return entry, nil
	}
	return s.backup.Get(ctx, id)
}

// Query searches log entries in both stores.
func (s *BackupLogStore) Query(ctx context.Context, filter *model.LogFilter, limit, offset int) ([]*model.LogEntry, error) {
	primaryResults, err := s.primary.Query(ctx, filter, limit, offset)
	if err != nil {
		return nil, err
	}

	if len(primaryResults) >= limit {
		return primaryResults, nil
	}

	// Supplement with backup results
	backupResults, err := s.backup.Query(ctx, filter, limit-len(primaryResults), 0)
	if err != nil {
		return primaryResults, nil
	}

	return append(primaryResults, backupResults...), nil
}

// Count counts entries in both stores.
func (s *BackupLogStore) Count(ctx context.Context, filter *model.LogFilter) (int64, error) {
	primaryCount, err := s.primary.Count(ctx, filter)
	if err != nil {
		return 0, err
	}

	backupCount, err := s.backup.Count(ctx, filter)
	if err != nil {
		return primaryCount, nil
	}

	return primaryCount + backupCount, nil
}

// Delete removes a log entry from either store.
func (s *BackupLogStore) Delete(ctx context.Context, id string) error {
	err := s.primary.Delete(ctx, id)
	if err == nil {
		return nil
	}

	err = s.backup.Delete(ctx, id)
	if err == nil {
		return nil
	}

	return fmt.Errorf("log entry not found in either store: %s", id)
}

// DeleteExpired removes old entries from both stores.
func (s *BackupLogStore) DeleteExpired(ctx context.Context, before time.Time) (int64, error) {
	primaryDeleted, _ := s.primary.DeleteExpired(ctx, before)
	backupDeleted, _ := s.backup.DeleteExpired(ctx, before)
	return primaryDeleted + backupDeleted, nil
}

// ListSources returns sources from both stores.
func (s *BackupLogStore) ListSources(ctx context.Context) ([]string, error) {
	sources1, err := s.primary.ListSources(ctx)
	if err != nil {
		return nil, err
	}

	sources2, err := s.backup.ListSources(ctx)
	if err != nil {
		return sources1, nil
	}

	sourceSet := make(map[string]bool)
	for _, s := range sources1 {
		sourceSet[s] = true
	}
	for _, s := range sources2 {
		sourceSet[s] = true
	}

	merged := make([]string, 0, len(sourceSet))
	for s := range sourceSet {
		merged = append(merged, s)
	}
	return merged, nil
}

// ListServices returns services from both stores.
func (s *BackupLogStore) ListServices(ctx context.Context) ([]string, error) {
	svcs1, err := s.primary.ListServices(ctx)
	if err != nil {
		return nil, err
	}

	svcs2, err := s.backup.ListServices(ctx)
	if err != nil {
		return svcs1, nil
	}

	svcSet := make(map[string]bool)
	for _, s := range svcs1 {
		svcSet[s] = true
	}
	for _, s := range svcs2 {
		svcSet[s] = true
	}

	merged := make([]string, 0, len(svcSet))
	for s := range svcSet {
		merged = append(merged, s)
	}
	return merged, nil
}

// Statistics returns statistics from both stores.
func (s *BackupLogStore) Statistics(ctx context.Context, from, to time.Time) (*LogStatistics, error) {
	stats1, err := s.primary.Statistics(ctx, from, to)
	if err != nil {
		return nil, err
	}

	stats2, err := s.backup.Statistics(ctx, from, to)
	if err != nil {
		return stats1, nil
	}

	// Merge statistics
	merged := &LogStatistics{
		ByLevel:   make(map[model.LogLevel]int64),
		BySource:  make(map[string]int64),
		ByService: make(map[string]int64),
	}

	merged.TotalCount = stats1.TotalCount + stats2.TotalCount
	for k, v := range stats1.ByLevel {
		merged.ByLevel[k] += v
	}
	for k, v := range stats2.ByLevel {
		merged.ByLevel[k] += v
	}
	for k, v := range stats1.BySource {
		merged.BySource[k] += v
	}
	for k, v := range stats2.BySource {
		merged.BySource[k] += v
	}

	if merged.TotalCount > 0 {
		errorCount := merged.ByLevel[model.LevelError] + merged.ByLevel[model.LevelFatal]
		merged.ErrorRate = float64(errorCount) / float64(merged.TotalCount)
	}

	return merged, nil
}

// HourlyBreakdown returns hourly breakdown from both stores.
func (s *BackupLogStore) HourlyBreakdown(ctx context.Context, from, to time.Time) ([]HourlyCount, error) {
	breakdown1, err := s.primary.HourlyBreakdown(ctx, from, to)
	if err != nil {
		return nil, err
	}

	breakdown2, err := s.backup.HourlyBreakdown(ctx, from, to)
	if err != nil {
		return breakdown1, nil
	}

	// Merge by hour+level key
	type key struct {
		hour  string
		level model.LogLevel
	}
	merged := make(map[key]int64)

	for _, h := range breakdown1 {
		merged[key{h.Hour, h.Level}] += h.Count
	}
	for _, h := range breakdown2 {
		merged[key{h.Hour, h.Level}] += h.Count
	}

	var result []HourlyCount
	for k, v := range merged {
		result = append(result, HourlyCount{
			Hour:  k.hour,
			Level: k.level,
			Count: v,
		})
	}

	return result, nil
}

// RegisterSource registers a valid source for log entries.
func (s *BackupLogStore) RegisterSource(source string) {
	s.primary.RegisterSource(source)
	s.backup.RegisterSource(source)
}

// Close releases resources.
func (s *BackupLogStore) Close() error {
	s.primary.Close()
	s.backup.Close()
	return nil
}

// Primary returns the primary store.
func (s *BackupLogStore) Primary() *MemoryLogStore {
	return s.primary
}

// Backup returns the backup store.
func (s *BackupLogStore) Backup() *MemoryLogStore {
	return s.backup
}
