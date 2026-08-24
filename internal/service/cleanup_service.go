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

// CleanupService handles periodic cleanup of expired data.
type CleanupService interface {
	// Start begins the periodic cleanup.
	Start(ctx context.Context) error
	// Stop halts the periodic cleanup.
	Stop()
	// CleanupOnce performs a single cleanup pass.
	CleanupOnce(ctx context.Context) error
	// SetFileStore sets the file persistence store for state saving.
	SetFileStore(fp *store.FilePersistence)
	// SetRuleStore sets the rule store for state saving.
	SetRuleStore(rs store.RuleStore)
}

// cleanupService is the default implementation of CleanupService.
type cleanupService struct {
	logStore   store.LogStore
	alertStore store.AlertStore
	ruleStore  store.RuleStore
	config     *config.Config
	logger     logger.Logger
	fileStore  *store.FilePersistence

	running bool
	stopCh  chan struct{}
}

// NewCleanupService creates a new CleanupService.
func NewCleanupService(ls store.LogStore, as store.AlertStore, cfg *config.Config, log logger.Logger) CleanupService {
	return &cleanupService{
		logStore:   ls,
		alertStore: as,
		config:     cfg,
		logger:     log.WithField("service", "cleanup"),
		stopCh:     make(chan struct{}),
	}
}

// Start begins the periodic cleanup.
func (s *cleanupService) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("cleanup service is already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})

	s.logger.Info("cleanup service started")

	go s.runLoop(ctx)
	return nil
}

// Stop halts the periodic cleanup.
func (s *cleanupService) Stop() {
	if !s.running {
		return
	}
	close(s.stopCh)
	s.running = false
	s.logger.Info("cleanup service stopped")
}

// CleanupOnce performs a single cleanup pass.
func (s *cleanupService) CleanupOnce(ctx context.Context) error {
	cutoff := time.Now().Add(-7 * 24 * time.Hour)

	logDeleted, err := s.logStore.DeleteExpired(ctx, cutoff)
	if err != nil {
		s.logger.Error("failed to delete expired logs", "error", err)
	} else {
		s.logger.Info("expired logs cleaned up", "count", logDeleted)
	}

	alertDeleted, err := s.alertStore.DeleteOld(ctx, cutoff)
	if err != nil {
		s.logger.Error("failed to delete old alerts", "error", err)
	} else {
		s.logger.Info("old alerts cleaned up", "count", alertDeleted)
	}

	if s.fileStore != nil {
		logs, err := s.logStore.Query(ctx, nil, 100000000, 0)
		if err != nil {
			s.logger.Error("failed to query logs for state save", "error", err)
		}

		var rules []*model.AlertRule
		if s.ruleStore != nil {
			allRules, err := s.ruleStore.List(ctx)
			if err != nil {
				s.logger.Error("failed to list rules for state save", "error", err)
			} else {
				rules = allRules
			}
		}

		var alerts []*model.AlertEvent
		allAlerts, err := s.alertStore.ListAll(ctx)
		if err != nil {
			s.logger.Error("failed to list alerts for state save", "error", err)
		} else {
			alerts = allAlerts
		}

		if err := s.fileStore.SaveState(ctx, logs, rules, alerts); err != nil {
			s.logger.Error("failed to save state after cleanup", "error", err)
			s.logger.Info("state save incomplete", "logs", len(logs), "rules", len(rules), "alerts", len(alerts))
			return fmt.Errorf("failed to save state after cleanup: %w", err)
		}

		s.logger.Info("state saved after cleanup", "logs", len(logs), "rules", len(rules), "alerts", len(alerts))
	}

	return nil
}

// SetFileStore sets the file persistence store for state saving.
func (s *cleanupService) SetFileStore(fp *store.FilePersistence) {
	s.fileStore = fp
}

// SetRuleStore sets the rule store for state saving.
func (s *cleanupService) SetRuleStore(rs store.RuleStore) {
	s.ruleStore = rs
}

// runLoop is the main cleanup loop.
func (s *cleanupService) runLoop(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := s.CleanupOnce(ctx); err != nil {
				s.logger.Error("cleanup failed", "error", err)
			}
		}
	}
}

// Verify imports
var _ = model.DefaultStatsRequest
