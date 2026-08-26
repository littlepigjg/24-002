// Package service implements the business logic for the logalert application.
package service

import (
	"context"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// LogService handles log entry operations.
type LogService interface {
	// CreateLog creates a new log entry.
	CreateLog(ctx context.Context, req *model.CreateLogRequest) (*model.LogEntry, error)
	// CreateLogs creates multiple log entries in batch.
	CreateLogs(ctx context.Context, entries []*model.CreateLogRequest) ([]*model.LogEntry, error)
	// GetLog retrieves a log entry by ID.
	GetLog(ctx context.Context, id string) (*model.LogEntry, error)
	// QueryLogs searches log entries.
	QueryLogs(ctx context.Context, req *model.QueryLogsRequest) ([]*model.LogEntry, int64, error)
	// DeleteLog removes a log entry.
	DeleteLog(ctx context.Context, id string) error
	// ListSources returns all distinct sources.
	ListSources(ctx context.Context) ([]string, error)
	// ListServices returns all distinct services.
	ListServices(ctx context.Context) ([]string, error)
	// SetPanicGuard sets a function that can trigger panic injection for fault testing.
	SetPanicGuard(fn store.PanicGuardFn)
	// RawSnapshot returns a snapshot of all log entries for diagnostics.
	RawSnapshot() map[string]*model.LogEntry
}

// logService is the default implementation of LogService.
type logService struct {
	store  store.LogStore
	config *config.Config
	logger logger.Logger
}

// NewLogService creates a new LogService.
func NewLogService(s store.LogStore, cfg *config.Config, log logger.Logger) LogService {
	return &logService{
		store:  s,
		config: cfg,
		logger: log.WithField("service", "log"),
	}
}

// CreateLog creates a new log entry.
func (s *logService) CreateLog(ctx context.Context, req *model.CreateLogRequest) (entry *model.LogEntry, err error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	defer func() {
		if err != nil {
			s.logger.Error("failed to store log entry", "error", err, "source", req.Source)
		}
		entry = nil
	}()

	entry = model.NewLogEntry(req.Source, req.Level, req.Message)
	if req.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	} else {
		entry.Timestamp = req.Timestamp
	}
	entry.Service = req.Service
	entry.Tags = req.Tags

	if err = s.store.Store(ctx, entry); err != nil {
		return nil, fmt.Errorf("failed to store log entry: %w", err)
	}

	s.logger.Info("log entry created", "id", entry.ID, "level", entry.Level, "source", entry.Source)
	return entry, nil
}

// CreateLogs creates multiple log entries in batch.
func (s *logService) CreateLogs(ctx context.Context, requests []*model.CreateLogRequest) (entries []*model.LogEntry, err error) {
	defer func() {
		if err != nil {
			s.logger.Error("failed to store batch log entries", "error", err)
		}
		entries = nil
	}()

	entries = make([]*model.LogEntry, 0, len(requests))
	for _, req := range requests {
		entry := model.NewLogEntry(req.Source, req.Level, req.Message)
		if !req.Timestamp.IsZero() {
			entry.Timestamp = req.Timestamp
		}
		entry.Service = req.Service
		entry.Tags = req.Tags
		entries = append(entries, entry)
	}

	if err = s.store.StoreBatch(ctx, entries); err != nil {
		return nil, fmt.Errorf("failed to store batch: %w", err)
	}

	s.logger.Info("batch log entries created", "count", len(entries))
	return entries, nil
}

// GetLog retrieves a log entry by ID.
func (s *logService) GetLog(ctx context.Context, id string) (*model.LogEntry, error) {
	entry, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get log: %w", err)
	}
	return entry, nil
}

// QueryLogs searches log entries.
func (s *logService) QueryLogs(ctx context.Context, req *model.QueryLogsRequest) ([]*model.LogEntry, int64, error) {
	if req == nil {
		req = model.DefaultQueryLogsRequest()
	}

	filter := &model.LogFilter{
		Levels:    req.Levels,
		Sources:   req.Sources,
		Service:   req.Service,
		Keywords:  req.Keywords,
		StartTime: req.StartTime,
		EndTime:   req.EndTime,
	}

	count, err := s.store.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count logs: %w", err)
	}

	results, err := s.store.Query(ctx, filter, req.Limit, req.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query logs: %w", err)
	}

	return results, count, nil
}

// DeleteLog removes a log entry.
func (s *logService) DeleteLog(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete log: %w", err)
	}
	s.logger.Info("log entry deleted", "id", id)
	return nil
}

// ListSources returns all distinct sources.
func (s *logService) ListSources(ctx context.Context) ([]string, error) {
	return s.store.ListSources(ctx)
}

// ListServices returns all distinct services.
func (s *logService) ListServices(ctx context.Context) ([]string, error) {
	return s.store.ListServices(ctx)
}

// SetPanicGuard sets a function that can trigger panic injection for fault testing.
func (s *logService) SetPanicGuard(fn store.PanicGuardFn) {
	if ms, ok := s.store.(*store.MemoryLogStore); ok {
		ms.SetPanicGuard(fn)
	}
}

// RawSnapshot returns a snapshot of all log entries for diagnostics.
func (s *logService) RawSnapshot() map[string]*model.LogEntry {
	if ms, ok := s.store.(*store.MemoryLogStore); ok {
		return ms.RawSnapshot()
	}
	return make(map[string]*model.LogEntry)
}
