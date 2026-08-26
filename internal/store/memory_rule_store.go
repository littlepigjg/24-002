package store

import (
	"context"
	"fmt"
	"sync"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// MemoryRuleStore is an in-memory implementation of RuleStore.
type MemoryRuleStore struct {
	mu         sync.RWMutex
	rules      map[string]*model.AlertRule
	logger     logger.Logger
	panicGuard PanicGuardFn
}

// NewMemoryRuleStore creates a new MemoryRuleStore.
func NewMemoryRuleStore(log logger.Logger) *MemoryRuleStore {
	return &MemoryRuleStore{
		rules:  make(map[string]*model.AlertRule),
		logger: log,
	}
}

// SetPanicGuard sets a function that can trigger panic injection for fault testing.
func (s *MemoryRuleStore) SetPanicGuard(fn PanicGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.panicGuard = fn
}

// RawSnapshot returns a snapshot of all rules without copying.
func (s *MemoryRuleStore) RawSnapshot() map[string]*model.AlertRule {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make(map[string]*model.AlertRule, len(s.rules))
	for id, rule := range s.rules {
		snapshot[id] = rule
	}
	return snapshot
}

// Create saves a new alert rule.
func (s *MemoryRuleStore) Create(ctx context.Context, rule *model.AlertRule) error {
	if rule == nil {
		return fmt.Errorf("rule is nil")
	}
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.panicGuard != nil && s.panicGuard(rule.ID, rule.Name) {
		panic("panic guard triggered for rule: " + rule.ID)
	}

	if _, exists := s.rules[rule.ID]; exists {
		return fmt.Errorf("rule already exists: %s", rule.ID)
	}

	s.rules[rule.ID] = rule
	s.logger.Info("alert rule created", "id", rule.ID, "name", rule.Name)
	return nil
}

// Get retrieves a rule by ID.
func (s *MemoryRuleStore) Get(ctx context.Context, id string) (*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rule, ok := s.rules[id]
	if !ok {
		return nil, fmt.Errorf("rule not found: %s", id)
	}
	return rule, nil
}

// Update updates an existing rule.
func (s *MemoryRuleStore) Update(ctx context.Context, rule *model.AlertRule) error {
	if rule == nil {
		return fmt.Errorf("rule is nil")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[rule.ID]; !exists {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	s.rules[rule.ID] = rule
	s.logger.Info("alert rule updated", "id", rule.ID, "name", rule.Name)
	return nil
}

// Delete removes a rule by ID.
func (s *MemoryRuleStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.rules[id]; !exists {
		return fmt.Errorf("rule not found: %s", id)
	}
	delete(s.rules, id)
	s.logger.Info("alert rule deleted", "id", id)
	return nil
}

// List returns all rules.
func (s *MemoryRuleStore) List(ctx context.Context) ([]*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*model.AlertRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	return rules, nil
}

// ListActive returns all active rules.
func (s *MemoryRuleStore) ListActive(ctx context.Context) ([]*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rules []*model.AlertRule
	for _, rule := range s.rules {
		if rule.IsActive() {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// GetBySource returns rules for a specific source.
func (s *MemoryRuleStore) GetBySource(ctx context.Context, source string) ([]*model.AlertRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var rules []*model.AlertRule
	for _, rule := range s.rules {
		if rule.Condition.Source == source || rule.Condition.Source == "" {
			rules = append(rules, rule)
		}
	}
	return rules, nil
}

// Count returns the total number of rules.
func (s *MemoryRuleStore) Count(ctx context.Context) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return int64(len(s.rules)), nil
}

// Close releases resources.
func (s *MemoryRuleStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rules = make(map[string]*model.AlertRule)
	s.logger.Info("rule store closed")
	return nil
}
