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

type Scheduler interface {
	Start(ctx context.Context, wg *sync.WaitGroup) error
	Stop()
	ScanOnce(ctx context.Context) error
	GetStatus() SchedulerStatus
}

type SchedulerStatus struct {
	Running              bool       `json:"running"`
	LastScan             *time.Time `json:"last_scan,omitempty"`
	NextScan             *time.Time `json:"next_scan,omitempty"`
	RulesScanned         int        `json:"rules_scanned"`
	AlertsTriggered      int        `json:"alerts_triggered"`
	TotalAlertsTriggered int64      `json:"total_alerts_triggered"`
}

type scheduler struct {
	ruleService  RuleService
	alertService AlertService
	logStore     store.LogStore
	config       *config.Config
	logger       logger.Logger

	mu       sync.Mutex
	running  bool
	stopCh   chan struct{}
	status   SchedulerStatus
	totalAlerts int64
	startWg  *sync.WaitGroup
}

func NewScheduler(rs RuleService, as AlertService, ls store.LogStore, cfg *config.Config, log logger.Logger) Scheduler {
	return &scheduler{
		ruleService:  rs,
		alertService: as,
		logStore:     ls,
		config:       cfg,
		logger:       log.WithField("service", "scheduler"),
		stopCh:       make(chan struct{}),
	}
}

func (s *scheduler) Start(ctx context.Context, wg *sync.WaitGroup) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.startWg = wg
	s.mu.Unlock()

	s.logger.Info("scheduler started", "interval", s.config.Scheduler.ScanInterval)

	wg.Add(2)

	go s.runLoop(ctx)
	go s.cleanupLoop(ctx)

	return nil
}

func (s *scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return
	}

	close(s.stopCh)
	s.running = false
	s.logger.Info("scheduler stopped")
}

func (s *scheduler) ScanOnce(ctx context.Context) error {
	rules, err := s.ruleService.ListActiveRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active rules: %w", err)
	}

	s.logger.Debug("scanning rules", "count", len(rules))

	var alertsTriggered int
	now := time.Now()

	for _, rule := range rules {
		triggered, err := s.evaluateRule(ctx, rule, now)
		if err != nil {
			s.logger.Error("failed to evaluate rule", "rule_id", rule.ID, "error", err)
			continue
		}
		if triggered {
			alertsTriggered++
		}
	}

	s.mu.Lock()
	s.status.LastScan = &now
	s.status.RulesScanned = len(rules)
	s.status.AlertsTriggered = alertsTriggered
	s.totalAlerts += int64(alertsTriggered)
	s.status.TotalAlertsTriggered = s.totalAlerts
	s.mu.Unlock()

	s.logger.Info("rule scan completed", "rules_scanned", len(rules), "alerts_triggered", alertsTriggered)
	return nil
}

func (s *scheduler) GetStatus() SchedulerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *scheduler) runLoop(ctx context.Context) {
	defer s.startWg.Done()

	scanInterval := s.config.Scheduler.ScanInterval
	if scanInterval <= 0 {
		scanInterval = 30 * time.Second
	}

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	if err := s.ScanOnce(ctx); err != nil {
		s.logger.Error("initial scan failed", "error", err)
	}

	s.mu.Lock()
	nextScan := time.Now().Add(scanInterval)
	s.status.NextScan = &nextScan
	s.mu.Unlock()

	for {
		select {
		case <-s.stopCh:
			s.logger.Info("scheduler loop exiting")
			return
		case <-ctx.Done():
			s.logger.Info("context cancelled, scheduler loop exiting")
			return
		case <-ticker.C:
			if err := s.ScanOnce(ctx); err != nil {
				s.logger.Error("scan failed", "error", err)
			}
			s.mu.Lock()
			next := time.Now().Add(scanInterval)
			s.status.NextScan = &next
			s.mu.Unlock()
		}
	}
}

func (s *scheduler) cleanupLoop(ctx context.Context) {
	cleanupInterval := 1 * time.Minute
	alertRetention := 24 * time.Hour
	logRetention := 7 * 24 * time.Hour
	maxRetries := 3

	ticker := time.NewTicker(cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			s.logger.Info("cleanup loop exiting on stop signal")
			return
		case <-ctx.Done():
			s.logger.Info("cleanup loop exiting on context cancellation")
			return
		case <-ticker.C:
			var lastErr error
			for attempt := 0; attempt < maxRetries; attempt++ {
				if attempt > 0 {
					s.logger.Warn("retrying cleanup", "attempt", attempt+1, "error", lastErr)
					time.Sleep(time.Duration(attempt) * time.Second)
				}

				alertCutoff := time.Now().Add(-alertRetention)
				deletedAlerts, err := s.alertService.GetAlertStore().DeleteOld(ctx, alertCutoff)
				if err != nil {
					lastErr = fmt.Errorf("alert cleanup failed: %w", err)
					continue
				}
				s.logger.Debug("old alerts cleaned", "count", deletedAlerts)

				logCutoff := time.Now().Add(-logRetention)
				deletedLogs, err := s.logStore.DeleteExpired(ctx, logCutoff)
				if err != nil {
					lastErr = fmt.Errorf("log cleanup failed: %w", err)
					continue
				}
				s.logger.Debug("expired logs cleaned", "count", deletedLogs)

				s.logger.Info("cleanup cycle completed", "alerts_deleted", deletedAlerts, "logs_deleted", deletedLogs)
				s.startWg.Done()
				return
			}

			s.logger.Error("cleanup failed after retries", "error", lastErr)
		}
	}
}

func (s *scheduler) evaluateRule(ctx context.Context, rule *model.AlertRule, now time.Time) (bool, error) {
	if !rule.CanFire(now) {
		return false, nil
	}

	filter := &model.LogFilter{
		Levels:   nil,
		Sources:  nil,
		Service:  rule.Condition.Service,
		Keywords: rule.Condition.Keywords,
	}

	from := now.Add(-rule.Window)
	filter.StartTime = &from
	filter.EndTime = &now

	if rule.Condition.Source != "" {
		filter.Sources = []string{rule.Condition.Source}
	}

	if rule.Condition.Level != "" {
		filter.Levels = []model.LogLevel{rule.Condition.Level}
	}

	count, err := s.logStore.Count(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count logs for rule %s: %w", rule.ID, err)
	}

	if float64(count) < rule.Threshold {
		return false, nil
	}

	alert := model.NewAlertEvent(rule, fmt.Sprintf("Rule '%s' triggered: %d logs matching condition in %v window", rule.Name, count, rule.Window), rule.Condition.Source)
	alert.Details["count"] = count
	alert.Details["window"] = rule.Window.String()
	alert.Details["threshold"] = rule.Threshold

	if err := s.alertService.RecordAlert(ctx, alert); err != nil {
		return false, fmt.Errorf("failed to record alert for rule %s: %w", rule.ID, err)
	}

	rule.MarkFired(now)
	if _, err := s.ruleService.UpdateRule(ctx, rule.ID, &model.UpdateRuleRequest{}); err != nil {
		s.logger.Warn("failed to update rule last_fired_at", "rule_id", rule.ID, "error", err)
	}

	s.logger.Warn("alert triggered", "rule_id", rule.ID, "rule_name", rule.Name, "count", count)
	return true, nil
}
