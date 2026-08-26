package logalert_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	ctx := context.Background()

	t.Run("MemoryLogStore", func(t *testing.T) {
		logStore := store.NewMemoryLogStore(1000, logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))

		logStore.SetFaultInjector(func(operation string) error {
			return errors.New("storage backend write failed")
		})

		entry := model.NewLogEntry("log-collector", model.LevelError, "connection timeout to downstream service")

		err := logStore.Store(ctx, entry)
		if err == nil {
			t.Fatal("expected error from Store with fault injection enabled")
		}

		done := make(chan struct{})
		go func() {
			logStore.Store(ctx, entry)
			close(done)
		}()

		select {
		case <-done:
			t.Log("GREEN（绿灯，缺陷已修复）")
		case <-time.After(2 * time.Second):
			t.Log("RED（红灯，缺陷未修复）")
			t.Fatal("Store blocked after fault injection error - lock leak detected")
		}
	})

	t.Run("MemoryAlertStore", func(t *testing.T) {
		alertStore := store.NewMemoryAlertStore(1000, logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))

		alertStore.SetFaultInjector(func(operation string) error {
			return errors.New("storage backend write failed")
		})

		rule := model.NewAlertRule("cpu-spike-detector", model.RuleCondition{Type: model.ConditionKeyword})
		alert := model.NewAlertEvent(rule, "CPU usage exceeded threshold", "monitoring-agent")

		err := alertStore.Record(ctx, alert)
		if err == nil {
			t.Fatal("expected error from Record with fault injection enabled")
		}

		done := make(chan struct{})
		go func() {
			alertStore.Record(ctx, alert)
			close(done)
		}()

		select {
		case <-done:
			t.Log("GREEN（绿灯，缺陷已修复）")
		case <-time.After(2 * time.Second):
			t.Log("RED（红灯，缺陷未修复）")
			t.Fatal("Record blocked after fault injection error - lock leak detected")
		}
	})
}
