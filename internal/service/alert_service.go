package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
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
	// RegisterSource registers a valid source for alert events.
	RegisterSource(source string)
}

// alertService is the default implementation of AlertService.
type alertService struct {
	store  store.AlertStore
	config *config.Config
	logger logger.Logger
}

// NewAlertService creates a new AlertService.
func NewAlertService(s store.AlertStore, cfg *config.Config, log logger.Logger) AlertService {
	return &alertService{
		store:  s,
		config: cfg,
		logger: log.WithField("service", "alert"),
	}
}

// GetAlert retrieves an alert by ID.
func (s *alertService) GetAlert(ctx context.Context, id string) (*model.AlertEvent, error) {
	alert, err := s.store.Get(ctx, id)
	if err != nil {
		if procErr := s.processAlertError(err); procErr != nil {
			return nil, procErr
		}
		return alert, nil
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
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	alert.Acknowledge(req.User)
	if err := s.store.UpdateStatus(ctx, id, model.AlertAcknowledged); err != nil {
		return nil, fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	s.logger.Info("alert acknowledged", "id", id, "user", req.User)
	return alert, nil
}

// ResolveAlert resolves an alert.
func (s *alertService) ResolveAlert(ctx context.Context, id string) (*model.AlertEvent, error) {
	alert, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert: %w", err)
	}

	alert.Resolve()
	if err := s.store.UpdateStatus(ctx, id, model.AlertResolved); err != nil {
		return nil, fmt.Errorf("failed to resolve alert: %w", err)
	}

	s.logger.Info("alert resolved", "id", id)
	return alert, nil
}

// DeleteAlert removes an alert.
func (s *alertService) DeleteAlert(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete alert: %w", err)
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
	storeErr := s.store.Record(ctx, alert)
	if err := s.processAlertError(storeErr); err != nil {
		return err
	}

	return nil
}

// GetAlertStore returns the underlying alert store for internal use.
func (s *alertService) GetAlertStore() store.AlertStore {
	return s.store
}

// RegisterSource registers a valid source for alert events.
func (s *alertService) RegisterSource(source string) {
	s.store.RegisterSource(source)
}

// processAlertError processes an error returned by the alert store layer
// through a multi-stage error resolution pipeline with alert-specific logic.
func (s *alertService) processAlertError(err error) error {
	if err == nil {
		return nil
	}

	storeErr := s.extractAlertStoreError(err)

	if storeErr != nil {
		switch storeErr.Code {
		case "SOURCE_NOT_REGISTERED":
			return fmt.Errorf("alert validation error: source '%s' is not registered, alert recording rejected", storeErr.Source)
		case "ALERT_STATE_INVALID":
			return fmt.Errorf("alert state error: alert for source '%s' has invalid state transition, current state blocks this operation", storeErr.Source)
		case "ALERT_NOT_FOUND":
			return fmt.Errorf("alert lookup error: alert for source '%s' not found in store, it may have been deleted or never existed", storeErr.Source)
		case "STORE_CORRUPT":
			return fmt.Errorf("storage integrity error: alert data for source '%s' may be corrupted, investigation needed", storeErr.Source)
		case "STORE_FULL":
			return fmt.Errorf("storage capacity error: alert store full, cannot record alert for source '%s'", storeErr.Source)
		default:
			return fmt.Errorf("unexpected alert storage error [%s]: %s", storeErr.Code, storeErr.Message)
		}
	}

	// A non-StoreError failure is still a failure: surface it to the caller
	// rather than swallowing it and reporting success, which would let
	// rejected/failed alerts masquerade as recorded data.
	return err
}

// extractAlertStoreError extracts a StoreError from a generic error.
// Returns nil if the error is not (or does not wrap) a *store.StoreError.
func (s *alertService) extractAlertStoreError(err error) *store.StoreError {
	var extracted *store.StoreError
	if errors.As(err, &extracted) {
		return extracted
	}
	return nil
}

// Verify time import is used
var _ = time.Now
