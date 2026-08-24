package logalert_test

import (
	"context"
	"fmt"
	"testing"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter())
	ctx := context.Background()

	t.Run("ListRules with many rules should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED（红灯，缺陷未修复）: List rules panicked: %v", r)
				t.Fail()
			}
		}()

		ruleStore := store.NewMemoryRuleStore(log)

		for i := 0; i < 5; i++ {
			rule := model.NewAlertRule(
				fmt.Sprintf("rule-%d", i),
				model.RuleCondition{
					Type:   model.ConditionCount,
					Level:  model.LevelError,
					Source: fmt.Sprintf("source-%d", i),
				},
			)
			if err := ruleStore.Create(ctx, rule); err != nil {
				t.Fatalf("failed to create rule %d: %v", i, err)
			}
		}

		rules, err := ruleStore.List(ctx)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: List rules returned error: %v", err)
			t.Fail()
			return
		}
		if len(rules) != 5 {
			t.Logf("RED（红灯，缺陷未修复）: Expected 5 rules, got %d", len(rules))
			t.Fail()
			return
		}
		t.Log("GREEN（绿灯，缺陷已修复）: List rules works correctly")
	})

	t.Run("QueryLogs with many entries should not panic", func(t *testing.T) {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("RED（红灯，缺陷未修复）: Query logs panicked: %v", r)
				t.Fail()
			}
		}()

		logStore := store.NewMemoryLogStore(1000, log)

		for i := 0; i < 5; i++ {
			entry := model.NewLogEntry(
				fmt.Sprintf("source-%d", i),
				model.LevelInfo,
				fmt.Sprintf("message-%d", i),
			)
			if err := logStore.Store(ctx, entry); err != nil {
				t.Fatalf("failed to store log entry %d: %v", i, err)
			}
		}

		results, err := logStore.Query(ctx, nil, 10, 0)
		if err != nil {
			t.Logf("RED（红灯，缺陷未修复）: Query logs returned error: %v", err)
			t.Fail()
			return
		}
		if len(results) != 5 {
			t.Logf("RED（红灯，缺陷未修复）: Expected 5 results, got %d", len(results))
			t.Fail()
			return
		}
		t.Log("GREEN（绿灯，缺陷已修复）: Query logs works correctly")
	})
}
