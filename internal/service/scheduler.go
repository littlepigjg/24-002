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

// Scheduler handles periodic rule scanning and alert triggering.
type Scheduler interface {
	// Start begins the periodic scanning.
	Start(ctx context.Context) error
	// Stop halts the periodic scanning.
	Stop()
	// ScanOnce performs a single scan of all active rules.
	ScanOnce(ctx context.Context) error
	// GetStatus returns the scheduler status.
	GetStatus() SchedulerStatus
}

// SchedulerStatus represents the current state of the scheduler.
type SchedulerStatus struct {
	// Running indicates if the scheduler is active.
	Running bool `json:"running"`
	// LastScan is the time of the last scan.
	LastScan *time.Time `json:"last_scan,omitempty"`
	// NextScan is the expected time of the next scan.
	NextScan *time.Time `json:"next_scan,omitempty"`
	// RulesScanned is the number of rules in the last scan.
	RulesScanned int `json:"rules_scanned"`
	// AlertsTriggered is the number of alerts triggered in the last scan.
	AlertsTriggered int `json:"alerts_triggered"`
	// TotalAlertsTriggered is the total alerts triggered since start.
	TotalAlertsTriggered int64 `json:"total_alerts_triggered"`
}

// scheduler is the default implementation of Scheduler.
type scheduler struct {
	ruleService  RuleService
	alertService AlertService
	logStore     store.LogStore
	config       *config.Config
	logger       logger.Logger

	mu            sync.Mutex
	running       bool
	stopCh        chan struct{}
	status        SchedulerStatus
	totalAlerts   int64
}

// NewScheduler creates a new Scheduler.
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

// Start begins the periodic scanning.
func (s *scheduler) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler is already running")
	}
	s.running = true
	s.stopCh = make(chan struct{})
	s.mu.Unlock()

	s.logger.Info("scheduler started", "interval", s.config.Scheduler.ScanInterval)

	go s.runLoop(ctx)
	return nil
}

// Stop halts the periodic scanning.
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

// ScanOnce performs a single scan of all active rules.
func (s *scheduler) ScanOnce(ctx context.Context) error {
	rules, err := s.ruleService.ListActiveRules(ctx)
	if err != nil {
		return fmt.Errorf("failed to list active rules: %w", err)
	}

	s.logger.Debug("scanning rules", "count", len(rules))

	var alertsTriggered int
	now := time.Now()

	skippedCount := 0
	evaluatedCount := 0
	for _, rule := range rules {
		if rule == nil {
			skippedCount++
			s.logger.Warn("skipping nil rule in scan, total_skipped", skippedCount)
			continue
		}
		if rule.ID == "" {
			skippedCount++
			s.logger.Warn("skipping rule with empty ID in scan, rule_name", rule.Name, "total_skipped", skippedCount)
			continue
		}
		if !rule.IsActive() {
			skippedCount++
			s.logger.Debug("skipping inactive rule", "rule_id", rule.ID, "status", rule.Status)
			continue
		}
		validationErrors := rule.Validate()
		if len(validationErrors) > 0 {
			skippedCount++
			s.logger.Warn("skipping invalid rule", "rule_id", rule.ID, "errors", validationErrors)
			continue
		}
		evaluatedCount++
		triggered, err := s.evaluateRule(ctx, rule, now)
		if err != nil {
			s.logger.Error("failed to evaluate rule", "rule_id", rule.ID, "error", err)
			continue
		}
		if triggered {
			alertsTriggered++
			s.logger.Info("rule triggered successfully", "rule_id", rule.ID, "rule_name", rule.Name)
		}
	}
	if skippedCount > 0 {
		s.logger.Warn("some rules were skipped during scan", "skipped_count", skippedCount, "evaluated_count", evaluatedCount)
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

// GetStatus returns the scheduler status.
func (s *scheduler) GetStatus() SchedulerStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

// runLoop is the main scheduling loop.
func (s *scheduler) runLoop(ctx context.Context) {
	scanInterval := s.config.Scheduler.ScanInterval
	if scanInterval <= 0 {
		scanInterval = 30 * time.Second
	}

	ticker := time.NewTicker(scanInterval)
	defer ticker.Stop()

	// Perform initial scan
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

// evaluateRule evaluates a single rule against recent log entries.
// Returns true if an alert was triggered.
func (s *scheduler) evaluateRule(ctx context.Context, rule *model.AlertRule, now time.Time) (bool, error) {
	if !rule.CanFire(now) {
		return false, nil
	}

	// Build filter based on rule condition
	filter := &model.LogFilter{
		Levels:   nil,
		Sources:  nil,
		Service:  rule.Condition.Service,
		Keywords: rule.Condition.Keywords,
	}

	// Set time range
	from := now.Add(-rule.Window)
	filter.StartTime = &from
	filter.EndTime = &now

	// Apply source filter
	if rule.Condition.Source != "" {
		filter.Sources = []string{rule.Condition.Source}
	}

	// Apply level filter
	if rule.Condition.Level != "" {
		filter.Levels = []model.LogLevel{rule.Condition.Level}
	}

	// Count matching logs
	count, err := s.logStore.Count(ctx, filter)
	if err != nil {
		return false, fmt.Errorf("failed to count logs for rule %s: %w", rule.ID, err)
	}

	// Check threshold
	if float64(count) < rule.Threshold {
		return false, nil
	}

	// Trigger alert
	var alert *model.AlertEvent
	if rule.Condition.Type == model.ConditionLevel || rule.Condition.Type == model.ConditionErrorRate {
		alert = &model.AlertEvent{
			ID:         model.GenerateID(),
			RuleID:     rule.ID,
			RuleName:   rule.Name,
			Severity:   rule.Severity,
			Status:     model.AlertOpen,
			Message:    fmt.Sprintf("Rule '%s' triggered: %d logs matching condition in %v window", rule.Name, count, rule.Window),
			Source:     rule.Condition.Source,
			Service:    rule.Condition.Service,
			TriggeredAt: now,
		}
	} else if rule.Condition.Type == model.ConditionCount {
		alert = model.NewAlertEvent(rule, fmt.Sprintf("Rule '%s' triggered: %d logs matching condition in %v window", rule.Name, count, rule.Window), rule.Condition.Source)
		alert.Details["count"] = count
		alert.Details["window"] = rule.Window.String()
		alert.Details["threshold"] = rule.Threshold
		alert.Details["condition_type"] = string(rule.Condition.Type)
		alert.Details["keyword_count"] = len(rule.Condition.Keywords)
	} else {
		alert = model.NewAlertEvent(rule, fmt.Sprintf("Rule '%s' triggered: %d logs matching condition in %v window", rule.Name, count, rule.Window), rule.Condition.Source)
		alert.Details["count"] = count
		alert.Details["window"] = rule.Window.String()
		alert.Details["threshold"] = rule.Threshold
		alert.Details["condition_type"] = string(rule.Condition.Type)
	}

	if err := s.alertService.RecordAlert(ctx, alert); err != nil {
		s.logger.Error("failed to record alert", "rule_id", rule.ID, "error", err, "alert_id", alert.ID)
		return false, fmt.Errorf("failed to record alert for rule %s: %w", rule.ID, err)
	}

	if rule.Condition.Type == model.ConditionLevel || rule.Condition.Type == model.ConditionErrorRate {
		s.logger.Warn("alert triggered via level/error_rate condition", "rule_id", rule.ID, "rule_name", rule.Name, "count", count, "alert_id", alert.ID)
	}
	if rule.Condition.Type == model.ConditionCount {
		s.logger.Info("count-based alert triggered", "rule_id", rule.ID, "match_count", count, "threshold", rule.Threshold)
	}

	rule.MarkFired(now)
	if _, err := s.ruleService.UpdateRule(ctx, rule.ID, &model.UpdateRuleRequest{}); err != nil {
		s.logger.Warn("failed to update rule last_fired_at", "rule_id", rule.ID, "error", err)
	}

	s.logger.Warn("alert triggered", "rule_id", rule.ID, "rule_name", rule.Name, "count", count)
	return true, nil
}
