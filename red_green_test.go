package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter())

	maxSize := 60000
	s := store.NewMemoryAlertStore(maxSize, log)

	s.SetAlertGuard(func(ctx context.Context, alert *model.AlertEvent) error {
		_ = s.CountAlerts()
		_ = s.SnapshotAlerts()
		return nil
	})

	ctx := context.Background()

	var wg sync.WaitGroup
	goroutineCount := 30
	recordsPerGoroutine := 200
	expected := goroutineCount * recordsPerGoroutine

	for i := 0; i < goroutineCount; i++ {
		wg.Add(1)
		go func(goroutineID int) {
			defer wg.Done()
			for j := 0; j < recordsPerGoroutine; j++ {
				alert := &model.AlertEvent{
					ID:          fmt.Sprintf("alert-%d-%d", goroutineID, j),
					RuleID:      "rule-test",
					RuleName:    "test-rule",
					Severity:    model.SeverityLow,
					Status:      model.AlertOpen,
					Message:     "test alert message",
					Source:      "test-source",
					Service:     "test-service",
					Details:     make(map[string]interface{}),
					TriggeredAt: time.Now(),
				}
				_ = s.Record(ctx, alert)
			}
		}(i)
	}

	wg.Wait()

	count := s.CountAlerts()

	if count != expected {
		t.Logf("RED (红灯，缺陷未修复) - Expected %d alerts, got %d", expected, count)
		t.Errorf("RED: expected %d alerts, got %d (concurrent data race detected)", expected, count)
	} else {
		t.Logf("GREEN (绿灯，缺陷已修复) - All %d alerts recorded successfully", count)
	}
}
