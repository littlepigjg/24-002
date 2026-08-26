package main

import (
	"context"
	"fmt"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// TestRedGreen verifies whether the nil pointer interface comparison bug
// is present. When the bug exists, errors from the store layer are silently
// lost and operations return success when they should fail.
//
// RED (bug present): Creating logs/alerts with unregistered sources succeeds
// GREEN (bug fixed): Creating logs/alerts with unregistered sources correctly fails
func TestRedGreen(t *testing.T) {
	logStore := store.NewMemoryLogStore(100, logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))
	logSvc := service.NewLogService(logStore, config.DefaultConfig(), logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))

	alertStore := store.NewMemoryAlertStore(100, logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))
	alertSvc := service.NewAlertService(alertStore, config.DefaultConfig(), logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))

	allPass := true
	ctx := context.Background()

	// Test 1: CreateLog with unregistered source should return error
	req := &model.CreateLogRequest{
		Level:   model.LevelInfo,
		Source:  "unregistered-source-for-log",
		Message: "test log message for red green check",
	}
	_, err := logSvc.CreateLog(ctx, req)
	if err != nil {
		t.Log("GREEN: CreateLog correctly rejected unregistered source")
	} else {
		t.Log("RED: CreateLog failed to reject unregistered source, operation silently succeeded")
		allPass = false
	}

	// Test 2: RecordAlert with unregistered source should return error
	alert := &model.AlertEvent{
		RuleID:   "rule-test-001",
		RuleName: "test-rule",
		Source:   "unregistered-source-for-alert",
		Message:  "test alert message for red green check",
		Severity: model.SeverityHigh,
		Status:   model.AlertOpen,
	}
	err = alertSvc.RecordAlert(ctx, alert)
	if err != nil {
		t.Log("GREEN: RecordAlert correctly rejected unregistered source")
	} else {
		t.Log("RED: RecordAlert failed to reject unregistered source, operation silently succeeded")
		allPass = false
	}

	// Test 3: Verify registered sources work correctly
	logSvc.RegisterSource("valid-log-source")
	req2 := &model.CreateLogRequest{
		Level:   model.LevelInfo,
		Source:  "valid-log-source",
		Message: "valid log message",
	}
	_, err2 := logSvc.CreateLog(ctx, req2)
	if err2 != nil {
		t.Log("RED: CreateLog incorrectly rejected registered source")
		allPass = false
	} else {
		t.Log("GREEN: CreateLog correctly accepted registered source")
	}

	alertSvc.RegisterSource("valid-alert-source")
	alert2 := &model.AlertEvent{
		RuleID:   "rule-test-002",
		RuleName: "test-rule-2",
		Source:   "valid-alert-source",
		Message:  "valid alert message",
		Severity: model.SeverityMedium,
		Status:   model.AlertOpen,
	}
	err3 := alertSvc.RecordAlert(ctx, alert2)
	if err3 != nil {
		t.Log("RED: RecordAlert incorrectly rejected registered source")
		allPass = false
	} else {
		t.Log("GREEN: RecordAlert correctly accepted registered source")
	}

	if !allPass {
		fmt.Println("=== RED (红灯，缺陷未修复) ===")
		t.Fail()
	} else {
		fmt.Println("=== GREEN (绿灯，缺陷已修复) ===")
	}
}
