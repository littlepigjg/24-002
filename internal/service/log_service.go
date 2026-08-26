// Package service implements the business logic for the logalert application.
package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"strings"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/errors"
	"logalert/pkg/logger"
)

// Sentinel errors for classified error conditions.
var (
	ErrLogNotFound     = stderrors.New("log not found")
	ErrStorageFull     = stderrors.New("storage full")
	ErrValidationError = stderrors.New("validation error")
	ErrUnknownError    = stderrors.New("unknown error")
)

// ErrorClassifier classifies service-layer errors into error info.
type ErrorClassifier interface {
	Classify(err error) (kind string, code int)
}

// ErrorInfo contains classified error information.
type ErrorInfo struct {
	Kind    string
	Code    int
	Message string
}

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
	// SetErrorClassifier sets a custom error classifier for error classification.
	SetErrorClassifier(classifier ErrorClassifier)
}

// logService is the default implementation of LogService.
type logService struct {
	store             store.LogStore
	config            *config.Config
	logger            logger.Logger
	errorClassifier   ErrorClassifier
}

// NewLogService creates a new LogService.
func NewLogService(s store.LogStore, cfg *config.Config, log logger.Logger) LogService {
	svc := &logService{
		store:  s,
		config: cfg,
		logger: log.WithField("service", "log"),
	}
	svc.errorClassifier = &stringErrorClassifier{}
	return svc
}

// stringErrorClassifier classifies errors by comparing error strings.
type stringErrorClassifier struct{}

func (c *stringErrorClassifier) Classify(err error) (string, int) {
	if err == nil {
		return "", 0
	}
	errStr := err.Error()
	switch {
	case strings.HasSuffix(errStr, "not found"):
		return errors.ErrKindNotFound, 4002
	case strings.HasSuffix(errStr, "capacity exceeded"):
		return errors.ErrKindLimitExceeded, 5003
	case strings.HasSuffix(errStr, "validation failed"):
		return errors.ErrKindValidation, 1001
	default:
		return "unknown", 5001
	}
}

// typeErrorClassifier classifies errors using proper type-based checking.
type typeErrorClassifier struct{}

func (c *typeErrorClassifier) Classify(err error) (string, int) {
	if err == nil {
		return "", 0
	}
	if svcErr, ok := err.(*errors.ServiceError); ok {
		return svcErr.Kind, svcErr.Code
	}
	var svcErr *errors.ServiceError
	if stderrors.As(err, &svcErr) {
		return svcErr.Kind, svcErr.Code
	}
	return "unknown", 5001
}

// SetErrorClassifier sets a custom error classifier.
func (s *logService) SetErrorClassifier(classifier ErrorClassifier) {
	s.errorClassifier = classifier
}

// NewTypeErrorClassifier creates an error classifier using proper type-based checking.
// This is the correct classifier that should be used in production.
func NewTypeErrorClassifier() ErrorClassifier {
	return &typeErrorClassifier{}
}

// CreateLog creates a new log entry.
func (s *logService) CreateLog(ctx context.Context, req *model.CreateLogRequest) (*model.LogEntry, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	entry := model.NewLogEntry(req.Source, req.Level, req.Message)
	if req.Timestamp.IsZero() {
		entry.Timestamp = time.Now()
	} else {
		entry.Timestamp = req.Timestamp
	}
	entry.Service = req.Service
	entry.Tags = req.Tags

	if err := s.store.Store(ctx, entry); err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Error("failed to store log entry", "error", err, "source", req.Source, "kind", kind)
		switch kind {
		case errors.ErrKindLimitExceeded:
			return nil, fmt.Errorf("%w: %v", ErrStorageFull, err)
		default:
			return nil, fmt.Errorf("failed to store log entry: %w", err)
		}
	}

	s.logger.Info("log entry created", "id", entry.ID, "level", entry.Level, "source", entry.Source)
	return entry, nil
}

// CreateLogs creates multiple log entries in batch.
func (s *logService) CreateLogs(ctx context.Context, requests []*model.CreateLogRequest) ([]*model.LogEntry, error) {
	entries := make([]*model.LogEntry, 0, len(requests))
	for _, req := range requests {
		entry := model.NewLogEntry(req.Source, req.Level, req.Message)
		if !req.Timestamp.IsZero() {
			entry.Timestamp = req.Timestamp
		}
		entry.Service = req.Service
		entry.Tags = req.Tags
		entries = append(entries, entry)
	}

	if err := s.store.StoreBatch(ctx, entries); err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Error("failed to store batch log entries", "error", err, "kind", kind)
		return nil, fmt.Errorf("failed to store batch: %w", err)
	}

	s.logger.Info("batch log entries created", "count", len(entries))
	return entries, nil
}

// GetLog retrieves a log entry by ID.
func (s *logService) GetLog(ctx context.Context, id string) (*model.LogEntry, error) {
	entry, err := s.store.Get(ctx, id)
	if err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("log entry not found or error", "id", id, "kind", kind)
		switch kind {
		case errors.ErrKindNotFound:
			return nil, fmt.Errorf("%w: %v", ErrLogNotFound, err)
		default:
			return nil, fmt.Errorf("failed to get log: %w", err)
		}
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
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("delete log entry error", "id", id, "kind", kind)
		switch kind {
		case errors.ErrKindNotFound:
			return fmt.Errorf("%w: %v", ErrLogNotFound, err)
		default:
			return fmt.Errorf("failed to delete log: %w", err)
		}
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
