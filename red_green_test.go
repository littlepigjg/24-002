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
	log := logger.NewLogger(logger.LogLevelError, logger.NewStdoutWriter())

	cfg := config.DefaultConfig()

	logStore := store.NewMemoryLogStore(500, log)
	for i := 0; i < 200; i++ {
		entry := model.NewLogEntry("test-source", model.LevelInfo, fmt.Sprintf("log message %d", i))
		logStore.Store(context.Background(), entry)
	}

	ruleStore := store.NewMemoryRuleStore(log)
	for i := 0; i < 50; i++ {
		rule := model.NewAlertRule(fmt.Sprintf("rule-%d", i), model.RuleCondition{
			Type:   model.ConditionCount,
			Source: "test-source",
		})
		ruleStore.Create(context.Background(), rule)
	}

	logService := service.NewLogService(logStore, cfg, log)
	ruleService := service.NewRuleService(ruleStore, cfg, log)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	_, _, err := logService.QueryLogs(ctx, &model.QueryLogsRequest{})

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel2()

	_, err2 := ruleService.ListActiveRules(ctx2)

	hasDefect := false

	if err == nil {
		hasDefect = true
		fmt.Println("QueryLogs: RED (红灯，缺陷未修复) - context timeout was ignored, operation completed despite deadline")
	} else if err == context.DeadlineExceeded {
		fmt.Println("QueryLogs: GREEN (绿灯，缺陷已修复) - context deadline properly respected")
	} else {
		fmt.Printf("QueryLogs: unexpected error: %v\n", err)
		hasDefect = true
	}

	if err2 == nil {
		hasDefect = true
		fmt.Println("ListActiveRules: RED (红灯，缺陷未修复) - context timeout was ignored, operation completed despite deadline")
	} else if err2 == context.DeadlineExceeded {
		fmt.Println("ListActiveRules: GREEN (绿灯，缺陷已修复) - context deadline properly respected")
	} else {
		fmt.Printf("ListActiveRules: unexpected error: %v\n", err2)
		hasDefect = true
	}

	if hasDefect {
		fmt.Println("RESULT: RED (红灯，缺陷未修复)")
		t.FailNow()
	}

	fmt.Println("RESULT: GREEN (绿灯，缺陷已修复)")
}
