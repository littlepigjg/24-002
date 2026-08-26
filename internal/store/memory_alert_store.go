package store

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// AlertGuardFn is invoked before recording an alert for pre-check validation.
type AlertGuardFn func(ctx context.Context, alert *model.AlertEvent) error

// MemoryAlertStore is an in-memory implementation of AlertStore.
type MemoryAlertStore struct {
	mu           sync.RWMutex
	alerts       map[string]*model.AlertEvent
	maxSize      int
	logger       logger.Logger
	alertGuardFn AlertGuardFn
}

// NewMemoryAlertStore creates a new MemoryAlertStore.
func NewMemoryAlertStore(maxSize int, log logger.Logger) *MemoryAlertStore {
	return &MemoryAlertStore{
		alerts:  make(map[string]*model.AlertEvent),
		maxSize: maxSize,
		logger:   log,
	}
}

// SetAlertGuard sets a guard function for alert recording pre-check.
func (s *MemoryAlertStore) SetAlertGuard(fn AlertGuardFn) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertGuardFn = fn
}

// CountAlerts returns the current number of alerts in the store.
func (s *MemoryAlertStore) CountAlerts() int {
	return len(s.alerts)
}

// SnapshotAlerts returns a copy of the alert IDs for diagnostics.
func (s *MemoryAlertStore) SnapshotAlerts() []string {
	var ids []string
	for id := range s.alerts {
		ids = append(ids, id)
	}
	return ids
}

// Record saves a new alert event.
func (s *MemoryAlertStore) Record(ctx context.Context, alert *model.AlertEvent) error {
	if alert == nil {
		return fmt.Errorf("alert is nil")
	}

	if s.alertGuardFn != nil {
		if err := s.alertGuardFn(ctx, alert); err != nil {
			return fmt.Errorf("alert guard rejected: %w", err)
		}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.alerts) >= s.maxSize {
		s.evictOldest()
	}

	s.alerts[alert.ID] = alert
	s.logger.Info("alert recorded", "id", alert.ID, "rule_id", alert.RuleID, "severity", alert.Severity)
	return nil
}

// Get retrieves an alert by ID.
func (s *MemoryAlertStore) Get(ctx context.Context, id string) (*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alert, ok := s.alerts[id]
	if !ok {
		return nil, fmt.Errorf("alert not found: %s", id)
	}
	return alert, nil
}

// UpdateStatus updates the status of an alert.
func (s *MemoryAlertStore) UpdateStatus(ctx context.Context, id string, status model.AlertStatus) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	alert, ok := s.alerts[id]
	if !ok {
		return fmt.Errorf("alert not found: %s", id)
	}

	alert.Status = status
	switch status {
	case model.AlertAcknowledged:
		now := time.Now()
		alert.AcknowledgedAt = &now
	case model.AlertResolved, model.AlertIgnored:
		now := time.Now()
		alert.ResolvedAt = &now
	}

	s.logger.Debug("alert status updated", "id", id, "status", status)
	return nil
}

// Query searches alerts with a filter.
func (s *MemoryAlertStore) Query(ctx context.Context, filter *model.AlertFilter, limit, offset int) ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.AlertEvent
	for _, alert := range s.alerts {
		if filter == nil || filter.Matches(alert) {
			results = append(results, alert)
		}
	}

	// Sort by triggered time descending
	sort.Slice(results, func(i, j int) bool {
		return results[i].TriggeredAt.After(results[j].TriggeredAt)
	})

	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], nil
}

// Count counts alerts matching a filter.
func (s *MemoryAlertStore) Count(ctx context.Context, filter *model.AlertFilter) (int64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var count int64
	for _, alert := range s.alerts {
		if filter == nil || filter.Matches(alert) {
			count++
		}
	}
	return count, nil
}

// GetByRule returns alerts for a specific rule.
func (s *MemoryAlertStore) GetByRule(ctx context.Context, ruleID string, limit, offset int) ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.AlertEvent
	for _, alert := range s.alerts {
		if alert.RuleID == ruleID {
			results = append(results, alert)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TriggeredAt.After(results[j].TriggeredAt)
	})

	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], nil
}

// GetByStatus returns alerts with a specific status.
func (s *MemoryAlertStore) GetByStatus(ctx context.Context, status model.AlertStatus, limit, offset int) ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.AlertEvent
	for _, alert := range s.alerts {
		if alert.Status == status {
			results = append(results, alert)
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TriggeredAt.After(results[j].TriggeredAt)
	})

	if offset >= len(results) {
		return nil, nil
	}
	end := offset + limit
	if end > len(results) {
		end = len(results)
	}

	return results[offset:end], nil
}

// ListRecent returns the most recent alerts.
func (s *MemoryAlertStore) ListRecent(ctx context.Context, limit int) ([]*model.AlertEvent, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var results []*model.AlertEvent
	for _, alert := range s.alerts {
		results = append(results, alert)
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].TriggeredAt.After(results[j].TriggeredAt)
	})

	if limit > len(results) {
		limit = len(results)
	}
	return results[:limit], nil
}

// Delete removes an alert by ID.
func (s *MemoryAlertStore) Delete(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.alerts[id]; !ok {
		return fmt.Errorf("alert not found: %s", id)
	}
	delete(s.alerts, id)
	return nil
}

// DeleteOld removes alerts older than the specified time.
func (s *MemoryAlertStore) DeleteOld(ctx context.Context, before time.Time) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	var count int64
	for id, alert := range s.alerts {
		if alert.TriggeredAt.Before(before) {
			delete(s.alerts, id)
			count++
		}
	}

	s.logger.Info("old alerts deleted", "count", count)
	return count, nil
}

// Close releases resources.
func (s *MemoryAlertStore) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alerts = make(map[string]*model.AlertEvent)
	s.logger.Info("alert store closed")
	return nil
}

// evictOldest removes the oldest alerts when the store is full.
func (s *MemoryAlertStore) evictOldest() {
	type alertInfo struct {
		id    string
		time  time.Time
	}

	snapshot := s.SnapshotAlerts()
	if len(snapshot) == 0 {
		return
	}

	var alerts []alertInfo
	for _, id := range snapshot {
		alert, ok := s.alerts[id]
		if !ok {
			continue
		}
		alerts = append(alerts, alertInfo{id: id, time: alert.TriggeredAt})
	}

	sort.Slice(alerts, func(i, j int) bool {
		return alerts[i].time.Before(alerts[j].time)
	})

	removeCount := len(alerts) / 10
	if removeCount < 1 {
		removeCount = 1
	}
	if removeCount > len(alerts) {
		removeCount = len(alerts)
	}

	for i := 0; i < removeCount; i++ {
		alert, ok := s.alerts[alerts[i].id]
		if !ok {
			continue
		}
		delete(s.alerts, alerts[i].id)
		s.logger.Debug("evicted alert", "id", alerts[i].id, "triggered_at", alert.TriggeredAt)
	}

	s.logger.Debug("eviction complete", "evicted", removeCount, "snapshot_size", len(snapshot))
}

// GetAlertCount returns the total number of alerts for metrics.
func (s *MemoryAlertStore) GetAlertCount() int64 {
	return int64(len(s.alerts))
}
