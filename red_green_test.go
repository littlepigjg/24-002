package main

import (
	"context"
	"fmt"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	// Setup
	cfg := config.DefaultConfig()
	logWriter := logger.NewDiscardWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWriter)

	// Create stores
	logStore := store.NewMemoryLogStore(10000, log)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(1000, log)

	// Create services
	logSvc := service.NewLogService(logStore, cfg, log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, cfg, log)
	scheduler := service.NewScheduler(ruleSvc, alertSvc, logStore, cfg, log)

	// Add log entries to satisfy rule condition (need 5 ERROR logs within 5 minutes)
	now := time.Now()
	for i := 0; i < 6; i++ {
		req := &model.CreateLogRequest{
			Timestamp: now.Add(-time.Duration(i) * time.Minute),
			Level:     model.LevelError,
			Source:    "test-service",
			Message:   fmt.Sprintf("error log %d", i),
			Service:   "test-app",
		}
		_, err := logSvc.CreateLog(context.Background(), req)
		if err != nil {
			t.Fatalf("failed to create log entry: %v", err)
		}
	}

	// Create an alert rule
	ruleReq := &model.CreateRuleRequest{
		Name: "test-error-rule",
		Condition: model.RuleCondition{
			Type:    model.ConditionCount,
			Level:   model.LevelError,
			Source:  "test-service",
			Service: "test-app",
		},
		Window:    5 * time.Minute,
		Threshold: 5,
		Severity:  model.SeverityHigh,
		Cooldown:  1 * time.Second,
	}

	rule, err := ruleSvc.CreateRule(context.Background(), ruleReq)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	// Verify rule was created
	if rule.ID == "" {
		t.Fatal("rule ID should not be empty")
	}

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Verify context is cancelled
	if err := ctx.Err(); err == nil {
		t.Fatal("context should be cancelled")
	}

	// Run scan with cancelled context - this is where the defect manifests
	err = scheduler.ScanOnce(ctx)

	// Check if alerts were triggered despite cancelled context
	alerts, alertCount, err := alertSvc.QueryAlerts(context.Background(), &model.QueryAlertsRequest{
		RuleIDs: []string{rule.ID},
		Limit:   10,
		Offset:  0,
	})

	if err != nil {
		t.Fatalf("failed to query alerts: %v", err)
	}

	// Determine pass/fail
	if alertCount > 0 {
		// DEFECT: Alerts were triggered despite cancelled context
		fmt.Println("RED (红灯，缺陷未修复)")
		fmt.Printf("Found %d alerts triggered with cancelled context - context timeout not properly propagated\n", alertCount)
		t.Errorf("context timeout propagation defect: %d alerts were triggered with cancelled context", alertCount)
	} else {
		// CORRECT: No alerts triggered with cancelled context
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		fmt.Println("Context timeout properly propagated - no alerts triggered with cancelled context")
		_ = alerts // suppress unused variable warning
	}
}
