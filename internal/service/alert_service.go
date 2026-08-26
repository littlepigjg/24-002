// Package service implements the business logic for the logalert application.
package service

import (
	"context"
	stderrors "errors"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// Sentinel errors for alert operations.
var (
	ErrAlertNotFound  = stderrors.New("alert not found")
	ErrStateConflict   = stderrors.New("state conflict")
)

// AlertService handles alert event operations.
type AlertService interface {
	// GetAlert retrieves an alert by ID.
	GetAlert(ctx context.Context, id string) (*model.AlertEvent, error)
	// QueryAlerts searches alert events.
	QueryAlerts(ctx context.Context, req *model.QueryAlertsRequest) ([]*model.AlertEvent, int64, error)
	// AcknowledgeAlert acknowledges an alert.
	AcknowledgeAlert(ctx context.Context, id string, req *model.AcknowledgeAlertRequest) (*model.AlertEvent, error)
	// ResolveAlert resolves an alert.
	ResolveAlert(ctx context.Context, id string) (*model.AlertEvent, error)
	// DeleteAlert removes an alert.
	DeleteAlert(ctx context.Context, id string) error
	// ListRecent returns the most recent alerts.
	ListRecent(ctx context.Context, limit int) ([]*model.AlertEvent, error)
	// GetAlertsByRule returns alerts for a specific rule.
	GetAlertsByRule(ctx context.Context, ruleID string, limit, offset int) ([]*model.AlertEvent, error)
	// GetOpenAlerts returns all open alerts.
	GetOpenAlerts(ctx context.Context) ([]*model.AlertEvent, error)
	// RecordAlert records a new alert event.
	RecordAlert(ctx context.Context, alert *model.AlertEvent) error
	// SetErrorClassifier sets a custom error classifier for error classification.
	SetErrorClassifier(classifier ErrorClassifier)
}

// alertService is the default implementation of AlertService.
type alertService struct {
	store           store.AlertStore
	config          *config.Config
	logger          logger.Logger
	errorClassifier ErrorClassifier
}

// NewAlertService creates a new AlertService.
func NewAlertService(s store.AlertStore, cfg *config.Config, log logger.Logger) AlertService {
	svc := &alertService{
		store:  s,
		config: cfg,
		logger: log.WithField("service", "alert"),
	}
	svc.errorClassifier = &stringErrorClassifier{}
	return svc
}

// SetErrorClassifier sets a custom error classifier.
func (s *alertService) SetErrorClassifier(classifier ErrorClassifier) {
	s.errorClassifier = classifier
}

// GetAlert retrieves an alert by ID.
func (s *alertService) GetAlert(ctx context.Context, id string) (*model.AlertEvent, error) {
	alert, err := s.store.Get(ctx, id)
	if err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("alert get error", "id", id, "kind", kind)
		switch kind {
		case "not_found":
			return nil, fmt.Errorf("%w: %v", ErrAlertNotFound, err)
		default:
			return nil, fmt.Errorf("failed to get alert: %w", err)
		}
	}
	return alert, nil
}

// QueryAlerts searches alert events.
func (s *alertService) QueryAlerts(ctx context.Context, req *model.QueryAlertsRequest) ([]*model.AlertEvent, int64, error) {
	if req == nil {
		req = model.DefaultQueryAlertsRequest()
	}

	filter := &model.AlertFilter{
		Statuses:   req.Statuses,
		Severities: req.Severities,
		RuleIDs:    req.RuleIDs,
		Source:     req.Source,
		StartTime:  req.StartTime,
		EndTime:    req.EndTime,
	}

	count, err := s.store.Count(ctx, filter)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count alerts: %w", err)
	}

	results, err := s.store.Query(ctx, filter, req.Limit, req.Offset)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query alerts: %w", err)
	}

	return results, count, nil
}

// AcknowledgeAlert acknowledges an alert.
func (s *alertService) AcknowledgeAlert(ctx context.Context, id string, req *model.AcknowledgeAlertRequest) (*model.AlertEvent, error) {
	alert, err := s.store.Get(ctx, id)
	if err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("acknowledge alert get error", "id", id, "kind", kind)
		switch kind {
		case "not_found":
			return nil, fmt.Errorf("%w: %v", ErrAlertNotFound, err)
		default:
			return nil, fmt.Errorf("failed to get alert: %w", err)
		}
	}

	// Update store status first before modifying the in-memory object
	// so that the store's state conflict check sees the current status.
	if err := s.store.UpdateStatus(ctx, id, model.AlertAcknowledged); err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("acknowledge alert update error", "id", id, "kind", kind)
		switch kind {
		case "state_conflict":
			return nil, fmt.Errorf("%w: %v", ErrStateConflict, err)
		default:
			return nil, fmt.Errorf("failed to acknowledge alert: %w", err)
		}
	}

	// Now mark the alert as acknowledged in memory
	alert.Acknowledge(req.User)
	s.logger.Info("alert acknowledged", "id", id, "user", req.User)
	return alert, nil
}

// ResolveAlert resolves an alert.
func (s *alertService) ResolveAlert(ctx context.Context, id string) (*model.AlertEvent, error) {
	alert, err := s.store.Get(ctx, id)
	if err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("resolve alert get error", "id", id, "kind", kind)
		switch kind {
		case "not_found":
			return nil, fmt.Errorf("%w: %v", ErrAlertNotFound, err)
		default:
			return nil, fmt.Errorf("failed to get alert: %w", err)
		}
	}

	// Update store status first before modifying the in-memory object
	if err := s.store.UpdateStatus(ctx, id, model.AlertResolved); err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("resolve alert update error", "id", id, "kind", kind)
		switch kind {
		case "state_conflict":
			return nil, fmt.Errorf("%w: %v", ErrStateConflict, err)
		default:
			return nil, fmt.Errorf("failed to resolve alert: %w", err)
		}
	}

	// Now mark the alert as resolved in memory
	alert.Resolve()
	s.logger.Info("alert resolved", "id", id)
	return alert, nil
}

// DeleteAlert removes an alert.
func (s *alertService) DeleteAlert(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		kind, _ := s.errorClassifier.Classify(err)
		s.logger.Warn("delete alert error", "id", id, "kind", kind)
		switch kind {
		case "not_found":
			return fmt.Errorf("%w: %v", ErrAlertNotFound, err)
		default:
			return fmt.Errorf("failed to delete alert: %w", err)
		}
	}
	s.logger.Info("alert deleted", "id", id)
	return nil
}

// ListRecent returns the most recent alerts.
func (s *alertService) ListRecent(ctx context.Context, limit int) ([]*model.AlertEvent, error) {
	if limit <= 0 {
		limit = 50
	}
	return s.store.ListRecent(ctx, limit)
}

// GetAlertsByRule returns alerts for a specific rule.
func (s *alertService) GetAlertsByRule(ctx context.Context, ruleID string, limit, offset int) ([]*model.AlertEvent, error) {
	return s.store.GetByRule(ctx, ruleID, limit, offset)
}

// GetOpenAlerts returns all open alerts.
func (s *alertService) GetOpenAlerts(ctx context.Context) ([]*model.AlertEvent, error) {
	return s.store.GetByStatus(ctx, model.AlertOpen, 100, 0)
}

// RecordAlert records a new alert event.
func (s *alertService) RecordAlert(ctx context.Context, alert *model.AlertEvent) error {
	return s.store.Record(ctx, alert)
}

// GetAlertStore returns the underlying alert store for internal use.
func (s *alertService) GetAlertStore() store.AlertStore {
	return s.store
}

// Verify time import is used
var _ = time.Now
