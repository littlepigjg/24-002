// Package store provides storage layer interfaces and implementations.
package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// FilePersistence handles saving/loading data to/from files.
type FilePersistence struct {
	mu      sync.Mutex
	dir     string
	logger  logger.Logger
}

// NewFilePersistence creates a new FilePersistence.
func NewFilePersistence(dir string, log logger.Logger) (*FilePersistence, error) {
	if dir == "" {
		dir = "./data"
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	return &FilePersistence{
		dir:    dir,
		logger: log.WithField("component", "file_persistence"),
	}, nil
}

// SaveLogs saves log entries to a JSON file.
func (fp *FilePersistence) SaveLogs(ctx context.Context, entries []*model.LogEntry) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	path := filepath.Join(fp.dir, "logs.json")
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write logs file: %w", err)
	}

	fp.logger.Debug("logs saved to file", "count", len(entries))
	return nil
}

// LoadLogs loads log entries from a JSON file.
func (fp *FilePersistence) LoadLogs(ctx context.Context) ([]*model.LogEntry, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	path := filepath.Join(fp.dir, "logs.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read logs file: %w", err)
	}

	var entries []*model.LogEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("failed to unmarshal logs: %w", err)
	}

	fp.logger.Debug("logs loaded from file", "count", len(entries))
	return entries, nil
}

// SaveRules saves rules to a JSON file.
func (fp *FilePersistence) SaveRules(ctx context.Context, rules []*model.AlertRule) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	path := filepath.Join(fp.dir, "rules.json")
	data, err := json.MarshalIndent(rules, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal rules: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write rules file: %w", err)
	}

	fp.logger.Debug("rules saved to file", "count", len(rules))
	return nil
}

// LoadRules loads rules from a JSON file.
func (fp *FilePersistence) LoadRules(ctx context.Context) ([]*model.AlertRule, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	path := filepath.Join(fp.dir, "rules.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read rules file: %w", err)
	}

	var rules []*model.AlertRule
	if err := json.Unmarshal(data, &rules); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rules: %w", err)
	}

	fp.logger.Debug("rules loaded from file", "count", len(rules))
	return rules, nil
}

// SaveAlerts saves alerts to a JSON file.
func (fp *FilePersistence) SaveAlerts(ctx context.Context, alerts []*model.AlertEvent) error {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	path := filepath.Join(fp.dir, "alerts.json")
	data, err := json.MarshalIndent(alerts, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal alerts: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write alerts file: %w", err)
	}

	fp.logger.Debug("alerts saved to file", "count", len(alerts))
	return nil
}

// LoadAlerts loads alerts from a JSON file.
func (fp *FilePersistence) LoadAlerts(ctx context.Context) ([]*model.AlertEvent, error) {
	fp.mu.Lock()
	defer fp.mu.Unlock()

	path := filepath.Join(fp.dir, "alerts.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read alerts file: %w", err)
	}

	var alerts []*model.AlertEvent
	if err := json.Unmarshal(data, &alerts); err != nil {
		return nil, fmt.Errorf("failed to unmarshal alerts: %w", err)
	}

	fp.logger.Debug("alerts loaded from file", "count", len(alerts))
	return alerts, nil
}

// SaveState saves the entire application state.
func (fp *FilePersistence) SaveState(ctx context.Context, logEntries []*model.LogEntry, rules []*model.AlertRule, alerts []*model.AlertEvent) error {
	if err := fp.SaveLogs(ctx, logEntries); err != nil {
		return err
	}
	if err := fp.SaveRules(ctx, rules); err != nil {
		return err
	}
	if err := fp.SaveAlerts(ctx, alerts); err != nil {
		return err
	}

	fp.logger.Info("state saved to file")
	return nil
}

// LoadState loads the entire application state.
func (fp *FilePersistence) LoadState(ctx context.Context) ([]*model.LogEntry, []*model.AlertRule, []*model.AlertEvent, error) {
	logs, err := fp.LoadLogs(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	rules, err := fp.LoadRules(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	alerts, err := fp.LoadAlerts(ctx)
	if err != nil {
		return nil, nil, nil, err
	}

	fp.logger.Info("state loaded from file", "logs", len(logs), "rules", len(rules), "alerts", len(alerts))
	return logs, rules, alerts, nil
}

// Now returns the current time (for consistent timestamps).
func Now() time.Time {
	return time.Now()
}
