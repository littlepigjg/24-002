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
	logWr := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWr)
	cfg := config.DefaultConfig()

	logStore := store.NewMemoryLogStore(1000, log)
	alertStore := store.NewMemoryAlertStore(1000, log)

	now := time.Now()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		entry := model.NewLogEntry("svc", model.LevelError, fmt.Sprintf("error-%d", i))
		entry.WithTimestamp(now)
		logStore.Store(ctx, entry)

		info := model.NewLogEntry("svc", model.LevelInfo, fmt.Sprintf("info-%d", i))
		info.WithTimestamp(now)
		logStore.Store(ctx, info)
	}

	rule := model.NewAlertRule("test-rule", model.RuleCondition{})

	for i := 0; i < 3; i++ {
		alert := model.NewAlertEvent(rule, "high alert", "svc")
		alert.Severity = model.SeverityHigh
		alert.TriggeredAt = now
		alertStore.Record(ctx, alert)
	}
	for i := 0; i < 7; i++ {
		alert := model.NewAlertEvent(rule, "low alert", "svc")
		alert.Severity = model.SeverityLow
		alert.TriggeredAt = now
		alertStore.Record(ctx, alert)
	}

	statsSvc := service.NewStatsService(logStore, alertStore, cfg, log)

	req := &model.StatsRequest{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
		GroupBy:   "hour",
	}

	result, err := statsSvc.GetErrorRateTrend(ctx, req)
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("GetErrorRateTrend returned error: %v", err)
	}

	if len(result) == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("expected at least one data point, got 0")
	}

	expectedRate := 0.725
	tolerance := 0.01

	for i, point := range result {
		diff := point.ErrorRate - expectedRate
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Errorf("point %d: expected error rate %.3f, got %.4f", i, expectedRate, point.ErrorRate)
			return
		}
		if point.TotalCount != 20 {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Errorf("point %d: expected total count 20, got %d", i, point.TotalCount)
			return
		}
		if point.ErrorCount != 10 {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Errorf("point %d: expected error count 10, got %d", i, point.ErrorCount)
			return
		}
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}

func TestRedGreenNoAlertStore(t *testing.T) {
	logWr := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWr)
	cfg := config.DefaultConfig()

	logStore := store.NewMemoryLogStore(1000, log)

	now := time.Now()
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		entry := model.NewLogEntry("svc", model.LevelError, fmt.Sprintf("err-%d", i))
		entry.WithTimestamp(now)
		logStore.Store(ctx, entry)

		info := model.NewLogEntry("svc", model.LevelInfo, fmt.Sprintf("info-%d", i))
		info.WithTimestamp(now)
		logStore.Store(ctx, info)
	}

	statsSvc := service.NewStatsService(logStore, nil, cfg, log)

	req := &model.StatsRequest{
		StartTime: now.Add(-1 * time.Hour),
		EndTime:   now.Add(1 * time.Hour),
		GroupBy:   "hour",
	}

	result, err := statsSvc.GetErrorRateTrend(ctx, req)
	if err != nil {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("GetErrorRateTrend returned error: %v", err)
	}

	if len(result) == 0 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("expected at least one data point, got 0")
	}

	expectedRate := 0.5
	tolerance := 0.001

	for i, point := range result {
		diff := point.ErrorRate - expectedRate
		if diff < 0 {
			diff = -diff
		}
		if diff > tolerance {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Errorf("point %d: expected error rate %.3f, got %.4f", i, expectedRate, point.ErrorRate)
			return
		}
		if point.TotalCount != 20 {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Errorf("point %d: expected total count 20, got %d", i, point.TotalCount)
			return
		}
		if point.ErrorCount != 10 {
			fmt.Printf("RED（红灯，缺陷未修复）\n")
			t.Errorf("point %d: expected error count 10, got %d", i, point.ErrorCount)
			return
		}
	}

	fmt.Println("GREEN（绿灯，缺陷已修复）")
}