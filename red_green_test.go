// Package service_test contains acceptance tests for error classification.
// RED = Defect present: errors are misclassified (go test fails)
// GREEN = Defect fixed: errors are correctly classified (go test passes)
package service_test

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	ctx := context.Background()
	cfg := config.DefaultConfig()
	log := logger.Default()
	allGreen := true

	// Test 1: log not found classification
	t.Run("log_not_found", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(100, log)
		logSvc := service.NewLogService(logStore, cfg, log)

		_, err := logSvc.GetLog(ctx, "nonexistent-id")
		if err == nil {
			t.Fatal("expected error")
		}

		if stderrors.Is(err, service.ErrLogNotFound) {
			fmt.Println("GREEN: log_not_found")
		} else {
			fmt.Println("RED: log_not_found - error not classified as ErrLogNotFound")
			allGreen = false
		}
	})

	// Test 2: delete log not found
	t.Run("delete_log_not_found", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(100, log)
		logSvc := service.NewLogService(logStore, cfg, log)

		err := logSvc.DeleteLog(ctx, "nonexistent-id")
		if err == nil {
			t.Fatal("expected error")
		}

		if stderrors.Is(err, service.ErrLogNotFound) {
			fmt.Println("GREEN: delete_log_not_found")
		} else {
			fmt.Println("RED: delete_log_not_found - error not classified as ErrLogNotFound")
			allGreen = false
		}
	})

	// Test 3: alert not found classification
	t.Run("alert_not_found", func(t *testing.T) {
		alertStore := store.NewMemoryAlertStore(100, log)
		alertSvc := service.NewAlertService(alertStore, cfg, log)

		_, err := alertSvc.GetAlert(ctx, "nonexistent-id")
		if err == nil {
			t.Fatal("expected error")
		}

		if stderrors.Is(err, service.ErrAlertNotFound) {
			fmt.Println("GREEN: alert_not_found")
		} else {
			fmt.Println("RED: alert_not_found - error not classified as ErrAlertNotFound")
			allGreen = false
		}
	})

	// Test 4: alert state conflict
	t.Run("alert_state_conflict", func(t *testing.T) {
		alertStore := store.NewMemoryAlertStore(100, log)
		alertSvc := service.NewAlertService(alertStore, cfg, log)

		rule := model.NewAlertRule("test-rule", model.RuleCondition{})
		alert := model.NewAlertEvent(rule, "test alert", "test-source")
		if err := alertSvc.RecordAlert(ctx, alert); err != nil {
			t.Fatalf("failed to record alert: %v", err)
		}

		if _, err := alertSvc.ResolveAlert(ctx, alert.ID); err != nil {
			t.Fatalf("failed to resolve alert: %v", err)
		}

		req := &model.AcknowledgeAlertRequest{User: "test-user"}
		_, err := alertSvc.AcknowledgeAlert(ctx, alert.ID, req)
		if err == nil {
			t.Fatal("expected state conflict error")
		}

		if stderrors.Is(err, service.ErrStateConflict) {
			fmt.Println("GREEN: alert_state_conflict")
		} else {
			fmt.Println("RED: alert_state_conflict - error not classified as ErrStateConflict")
			allGreen = false
		}
	})

	// Test 5: storage full classification
	t.Run("storage_full", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1, log)
		logSvc := service.NewLogService(logStore, cfg, log)

		req1 := &model.CreateLogRequest{Source: "test", Level: model.LevelInfo, Message: "msg1"}
		if _, err := logSvc.CreateLog(ctx, req1); err != nil {
			t.Fatalf("first create failed: %v", err)
		}

		req2 := &model.CreateLogRequest{Source: "test", Level: model.LevelInfo, Message: "msg2"}
		_, err := logSvc.CreateLog(ctx, req2)
		if err == nil {
			t.Fatal("expected storage full error")
		}

		if stderrors.Is(err, service.ErrStorageFull) {
			fmt.Println("GREEN: storage_full")
		} else {
			fmt.Println("RED: storage_full - error not classified as ErrStorageFull")
			allGreen = false
		}
	})

	if !allGreen {
		t.Fatal("RED（红灯，缺陷未修复）：存在错误类型分类失败")
	}
	fmt.Println("GREEN（绿灯，缺陷已修复）：所有错误类型分类正确")
}
