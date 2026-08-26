package logalert_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	log := logger.NewLogger(logger.LogLevelError, logger.NewStdoutWriter())

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	logSvc := service.NewLogService(logStore, cfg, log)

	ruleStore := store.NewMemoryRuleStore(log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)

	ctx := context.Background()
	hasDefect := false

	// Test 1: CreateLog should return the created entry
	req1 := &model.CreateLogRequest{
		Level:   model.LevelInfo,
		Source:  "test-service",
		Message: "test log entry 1",
		Service: "test-svc",
	}
	entry1, err1 := logSvc.CreateLog(ctx, req1)
	if err1 != nil {
		t.Fatalf("CreateLog returned error: %v", err1)
	}
	snapshot := logStore.RawSnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 entry in store, got %d", len(snapshot))
	}
	if entry1 == nil {
		hasDefect = true
		fmt.Println("Test 1: CreateLog returned nil but entry exists in store - DEFECT DETECTED")
	} else {
		fmt.Println("Test 1: CreateLog returned entry correctly - PASS")
		if entry1.ID == "" {
			t.Errorf("returned entry has empty ID")
		}
		if entry1.Source != req1.Source {
			t.Errorf("returned entry source = %s, want %s", entry1.Source, req1.Source)
		}
	}

	// Test 2: CreateRule should return the created rule
	req2 := &model.CreateRuleRequest{
		Name:      "test-rule",
		Condition: model.RuleCondition{Type: model.ConditionLevel, Level: model.LevelError, Source: "test-source"},
		Window:    300000000000,
		Threshold: 3,
		Severity:  model.SeverityHigh,
		Cooldown:  300000000000,
	}
	rule1, err2 := ruleSvc.CreateRule(ctx, req2)
	if err2 != nil {
		t.Fatalf("CreateRule returned error: %v", err2)
	}
	ruleSnapshot := ruleStore.RawSnapshot()
	if len(ruleSnapshot) != 1 {
		t.Fatalf("expected 1 rule in store, got %d", len(ruleSnapshot))
	}
	if rule1 == nil {
		hasDefect = true
		fmt.Println("Test 2: CreateRule returned nil but rule exists in store - DEFECT DETECTED")
	} else {
		fmt.Println("Test 2: CreateRule returned rule correctly - PASS")
		if rule1.ID == "" {
			t.Errorf("returned rule has empty ID")
		}
		if rule1.Name != req2.Name {
			t.Errorf("returned rule name = %s, want %s", rule1.Name, req2.Name)
		}
	}

	// Test 3: CreateLogs (batch) should return the created entries
	reqs := []*model.CreateLogRequest{
		{Level: model.LevelWarn, Source: "batch-svc", Message: "batch msg 1", Service: "batch"},
		{Level: model.LevelError, Source: "batch-svc", Message: "batch msg 2", Service: "batch"},
	}
	entries, err3 := logSvc.CreateLogs(ctx, reqs)
	if err3 != nil {
		t.Fatalf("CreateLogs returned error: %v", err3)
	}
	snapshot2 := logStore.RawSnapshot()
	if len(snapshot2) < 3 {
		t.Fatalf("expected at least 3 entries in store, got %d", len(snapshot2))
	}
	if entries == nil {
		hasDefect = true
		fmt.Println("Test 3: CreateLogs returned nil but entries exist in store - DEFECT DETECTED")
	} else {
		fmt.Println("Test 3: CreateLogs returned entries correctly - PASS")
		if len(entries) != 2 {
			t.Errorf("returned entries count = %d, want 2", len(entries))
		}
	}

	// Final verdict
	if hasDefect {
		fmt.Println("\n========================================")
		fmt.Println("RED (红灯，缺陷未修复)")
		fmt.Println("========================================")
		os.Exit(1)
	} else {
		fmt.Println("\n========================================")
		fmt.Println("GREEN (绿灯，缺陷已修复)")
		fmt.Println("========================================")
	}
}