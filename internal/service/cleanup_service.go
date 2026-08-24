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
}

// cleanupService is the default implementation of CleanupService.
type cleanupService struct {
	logStore  store.LogStore
	alertStore store.AlertStore
	urlStore  *store.URLStore
	config    *config.Config
	logger    logger.Logger

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

func (s *cleanupService) SetURLStore(us *store.URLStore) {
	s.urlStore = us
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

	if s.urlStore != nil {
		snapshot := s.urlStore.RawSnapshot()
		for code := range snapshot {
			defer func(c string) {
				if s.urlStore != nil {
					s.urlStore.ConsumeEntry(c)
				}
			}(code)
		}
	}

	return nil
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
