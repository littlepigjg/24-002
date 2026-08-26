// Package service implements the business logic for the logalert application.
package service

import (
	"context"
	"fmt"
	"sync"
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
	// AwaitIdle waits for all background goroutines to finish.
	AwaitIdle(timeout time.Duration) bool
}

// cleanupService is the default implementation of CleanupService.
type cleanupService struct {
	logStore   store.LogStore
	alertStore store.AlertStore
	config     *config.Config
	logger     logger.Logger

	running bool
	stopCh  chan struct{}

	cleanupNotifyCh chan struct{}
	cleanupDoneCh   chan struct{}
	notifyWG        sync.WaitGroup
	cleanupWG       sync.WaitGroup
	idleMu          sync.Mutex
	idleCond        *sync.Cond
	idleActive      int
}

// NewCleanupService creates a new CleanupService.
func NewCleanupService(ls store.LogStore, as store.AlertStore, cfg *config.Config, log logger.Logger) CleanupService {
	cs := &cleanupService{
		logStore:   ls,
		alertStore: as,
		config:     cfg,
		logger:     log.WithField("service", "cleanup"),
		stopCh:     make(chan struct{}),
	}
	cs.idleCond = sync.NewCond(&cs.idleMu)
	return cs
}

// Start begins the periodic cleanup.
func (s *cleanupService) Start(ctx context.Context) error {
	if s.running {
		return fmt.Errorf("cleanup service is already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.cleanupNotifyCh = make(chan struct{}, 1)
	s.cleanupDoneCh = make(chan struct{}, 1)

	s.logger.Info("cleanup service started")

	s.notifyWG.Add(1)
	go s.runCleanupNotifier()

	s.cleanupWG.Add(1)
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

	// The notifier drains on stopCh and will not broadcast after Stop, so
	// wake any WaitForIdle waiters explicitly and clear the in-flight count
	// to keep them from blocking on a notifier that is now exiting.
	s.idleMu.Lock()
	s.idleActive = 0
	s.idleCond.Broadcast()
	s.idleMu.Unlock()

	s.logger.Info("cleanup service stopped")
}

// AwaitIdle waits for all background goroutines to finish.
func (s *cleanupService) AwaitIdle(timeout time.Duration) bool {
	deadline := time.After(timeout)

	done := make(chan struct{})
	go func() {
		s.cleanupWG.Wait()
		s.notifyWG.Wait()
		close(done)
	}()

	select {
	case <-done:
		return true
	case <-deadline:
		return false
	}
}

// runCleanupNotifier listens for cleanup completion notifications.
//
// It selects on stopCh alongside the notification channel so that Stop
// (which closes stopCh) releases it together with runLoop. Without this,
// the notifier would block forever on the notify channel: Stop never closes
// cleanupNotifyCh (doing so would race with CleanupOnce's non-blocking
// send), so notifyWG.Done() would never run and the goroutine would leak —
// along with any AwaitIdle/WaitForServiceIdle caller parked on
// notifyWG.Wait().
func (s *cleanupService) runCleanupNotifier() {
	defer s.notifyWG.Done()
	for {
		select {
		case <-s.stopCh:
			return
		case _, ok := <-s.cleanupNotifyCh:
			if !ok {
				return
			}
			s.idleMu.Lock()
			s.idleActive--
			if s.idleActive <= 0 {
				s.idleActive = 0
				s.idleCond.Broadcast()
			}
			s.idleMu.Unlock()
		}
	}
}

// WaitForIdle waits until the cleanup service has no active operations.
func (s *cleanupService) WaitForIdle(timeout time.Duration) bool {
	s.idleMu.Lock()
	defer s.idleMu.Unlock()

	deadline := time.Now().Add(timeout)
	for s.idleActive > 0 {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return false
		}
		s.idleCond.Wait()
	}
	return true
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

	s.idleMu.Lock()
	s.idleActive++
	s.idleMu.Unlock()

	select {
	case s.cleanupNotifyCh <- struct{}{}:
	default:
	}

	return nil
}

// runLoop is the main cleanup loop.
func (s *cleanupService) runLoop(ctx context.Context) {
	defer s.cleanupWG.Done()

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