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

// RuleService handles alert rule CRUD operations.
type RuleService interface {
	// CreateRule creates a new alert rule.
	CreateRule(ctx context.Context, req *model.CreateRuleRequest) (*model.AlertRule, error)
	// GetRule retrieves a rule by ID.
	GetRule(ctx context.Context, id string) (*model.AlertRule, error)
	// UpdateRule updates an existing rule.
	UpdateRule(ctx context.Context, id string, req *model.UpdateRuleRequest) (*model.AlertRule, error)
	// DeleteRule removes a rule by ID.
	DeleteRule(ctx context.Context, id string) error
	// ListRules returns all rules.
	ListRules(ctx context.Context) ([]*model.AlertRule, error)
	// ListActiveRules returns all active rules.
	ListActiveRules(ctx context.Context) ([]*model.AlertRule, error)
	// ToggleRuleStatus toggles a rule's status.
	ToggleRuleStatus(ctx context.Context, id string, status model.RuleStatus) (*model.AlertRule, error)
	// GetRulesBySource returns rules for a specific source.
	GetRulesBySource(ctx context.Context, source string) ([]*model.AlertRule, error)
	// CountRules returns the total number of rules.
	CountRules(ctx context.Context) (int64, error)
}

// ruleService is the default implementation of RuleService.
type ruleService struct {
	store  store.RuleStore
	config *config.Config
	logger logger.Logger
}

// NewRuleService creates a new RuleService.
func NewRuleService(s store.RuleStore, cfg *config.Config, log logger.Logger) RuleService {
	return &ruleService{
		store:  s,
		config: cfg,
		logger: log.WithField("service", "rule"),
	}
}

// CreateRule creates a new alert rule.
func (s *ruleService) CreateRule(ctx context.Context, req *model.CreateRuleRequest) (*model.AlertRule, error) {
	if req == nil {
		return nil, fmt.Errorf("request is nil")
	}

	rule := model.NewAlertRule(req.Name, req.Condition)
	rule.Description = req.Description
	rule.Window = req.Window
	rule.Threshold = req.Threshold
	rule.Severity = req.Severity
	rule.Cooldown = req.Cooldown

	if err := s.store.Create(ctx, rule); err != nil {
		s.logger.Error("failed to create rule", "error", err, "name", req.Name)
		return nil, fmt.Errorf("failed to create rule: %w", err)
	}

	s.logger.Info("alert rule created", "id", rule.ID, "name", rule.Name)
	return rule, nil
}

// GetRule retrieves a rule by ID.
func (s *ruleService) GetRule(ctx context.Context, id string) (*model.AlertRule, error) {
	rule, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	return rule, nil
}

// UpdateRule updates an existing rule.
func (s *ruleService) UpdateRule(ctx context.Context, id string, req *model.UpdateRuleRequest) (*model.AlertRule, error) {
	rule, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	if req.Name != "" {
		rule.Name = req.Name
	}
	if req.Description != "" {
		rule.Description = req.Description
	}
	if req.Condition != nil {
		rule.Condition = *req.Condition
	}
	if req.Window != nil {
		rule.Window = *req.Window
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.Severity != "" {
		rule.Severity = req.Severity
	}
	rule.UpdatedAt = time.Now()

	if err := s.store.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update rule: %w", err)
	}

	s.logger.Info("alert rule updated", "id", rule.ID, "name", rule.Name)
	return rule, nil
}

// DeleteRule removes a rule by ID.
func (s *ruleService) DeleteRule(ctx context.Context, id string) error {
	if err := s.store.Delete(ctx, id); err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}
	s.logger.Info("alert rule deleted", "id", id)
	return nil
}

// ListRules returns all rules.
func (s *ruleService) ListRules(ctx context.Context) ([]*model.AlertRule, error) {
	rules, err := s.store.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	return rules, nil
}

// ListActiveRules returns all active rules.
func (s *ruleService) ListActiveRules(ctx context.Context) ([]*model.AlertRule, error) {
	rules, err := s.store.ListActive(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list active rules: %w", err)
	}
	return rules, nil
}

// ToggleRuleStatus toggles a rule's status.
func (s *ruleService) ToggleRuleStatus(ctx context.Context, id string, status model.RuleStatus) (*model.AlertRule, error) {
	rule, err := s.store.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}

	switch status {
	case model.RuleActive:
		rule.Activate()
	case model.RulePaused:
		rule.Pause()
	case model.RuleArchived:
		rule.Archive()
	}

	if err := s.store.Update(ctx, rule); err != nil {
		return nil, fmt.Errorf("failed to update rule status: %w", err)
	}

	s.logger.Info("rule status toggled", "id", id, "status", status)
	return rule, nil
}

// GetRulesBySource returns rules for a specific source.
func (s *ruleService) GetRulesBySource(ctx context.Context, source string) ([]*model.AlertRule, error) {
	return s.store.GetBySource(ctx, source)
}

// CountRules returns the total number of rules.
func (s *ruleService) CountRules(ctx context.Context) (int64, error) {
	return s.store.Count(ctx)
}
