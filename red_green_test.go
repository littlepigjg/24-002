package logalert

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
	cfg := config.DefaultConfig()
	log := logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter())

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)

	logSvc := service.NewLogService(logStore, cfg, log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, cfg, log)
	scheduler := service.NewScheduler(ruleSvc, alertSvc, logStore, cfg, log)

	ctx := context.Background()

	createRuleReq := &model.CreateRuleRequest{
		Name: "error-level-rule",
		Condition: model.RuleCondition{
			Type:   model.ConditionLevel,
			Level:  model.LevelError,
			Source: "",
		},
		Window:   5 * time.Minute,
		Threshold: 2,
		Severity: model.SeverityHigh,
		Cooldown:  1 * time.Second,
	}

	_, err := ruleSvc.CreateRule(ctx, createRuleReq)
	if err != nil {
		t.Fatalf("failed to create rule: %v", err)
	}

	for i := 0; i < 2; i++ {
		logReq := &model.CreateLogRequest{
			Level:   model.LevelError,
			Source:  "test-service",
			Message: fmt.Sprintf("error message %d", i),
		}
		_, err := logSvc.CreateLog(ctx, logReq)
		if err != nil {
			t.Fatalf("failed to create log entry %d: %v", i, err)
		}
	}

	var panicErr interface{}
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicErr = r
			}
		}()
		_ = scheduler.ScanOnce(ctx)
	}()

	if panicErr != nil {
		t.Errorf("RED (红灯，缺陷未修复) - panic detected: %v", panicErr)
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}