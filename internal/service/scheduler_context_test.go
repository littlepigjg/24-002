package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// setupSchedulerForContextTests wires a real scheduler against in-memory
// rule/alert stores and a log store whose Count call is artificially slow,
// so we can assert the scheduler aborts (instead of firing alerts) once the
// context is cancelled or expires.
func setupSchedulerForContextTests(t *testing.T, delay time.Duration) (Scheduler, RuleService, store.AlertStore, *store.MemoryLogStore) {
	t.Helper()
	log := logger.NewLogger(logger.LogLevelWarn, logger.NewStdoutWriter())
	cfg := config.DefaultConfig()

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	logStore.SetProcessingDelay(delay)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)

	rs := NewRuleService(ruleStore, cfg, log)
	as := NewAlertService(alertStore, cfg, log)
	sched := NewScheduler(rs, as, logStore, cfg, log)
	return sched, rs, alertStore, logStore
}

func seedMatchingRule(t *testing.T, rs RuleService) *model.AlertRule {
	t.Helper()
	rule, err := rs.CreateRule(context.Background(), &model.CreateRuleRequest{
		Name: "count-rule",
		Condition: model.RuleCondition{
			Type:     model.ConditionCount,
			Level:    model.LevelError,
			MinCount: 1,
		},
		Window:    time.Minute,
		Threshold: 1, // any matching log triggers it
		Cooldown:  0,
	})
	if err != nil {
		t.Fatalf("create rule: %v", err)
	}
	return rule
}

// TestSchedulerScanOnceAbortsOnCancelledContext verifies that ScanOnce stops
// as soon as it observes a cancelled context rather than churning through the
// rule (and its slow store.Count) for a caller that is already gone.
func TestSchedulerScanOnceAbortsOnCancelledContext(t *testing.T) {
	sched, rs, _, logStore := setupSchedulerForContextTests(t, 50*time.Millisecond)
	seedMatchingRule(t, rs)
	if err := logStore.Store(context.Background(), model.NewLogEntry("svc-a", model.LevelError, "boom")); err != nil {
		t.Fatalf("seed log: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // already cancelled before we scan

	start := time.Now()
	err := sched.ScanOnce(ctx)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("ScanOnce returned err=%v; want context.Canceled", err)
	}
	// The scheduler must bail out promptly instead of waiting out the 50ms
	// processing delay inside the store Count call.
	if elapsed > 30*time.Millisecond {
		t.Fatalf("ScanOnce took %v on a cancelled context; want to abort promptly", elapsed)
	}
}

// TestSchedulerScanOnceRespectsExpiryDuringCount exercises the case where the
// context expires *while* the store is doing its (delayed) work: the scan
// should fail with DeadlineExceeded, and crucially must not have recorded an
// alert after the caller's deadline elapsed.
func TestSchedulerScanOnceRespectsExpiryDuringCount(t *testing.T) {
	sched, rs, alertStore, logStore := setupSchedulerForContextTests(t, 50*time.Millisecond)
	seedMatchingRule(t, rs)
	if err := logStore.Store(context.Background(), model.NewLogEntry("svc-a", model.LevelError, "boom")); err != nil {
		t.Fatalf("seed log: %v", err)
	}

	// Deadline shorter than the 50ms store delay so it expires mid-Count.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	err := sched.ScanOnce(ctx)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("ScanOnce returned err=%v; want context.DeadlineExceeded", err)
	}

	// No alert should have been persisted for the dead caller.
	count, qerr := alertStore.Count(context.Background(), nil)
	if qerr != nil {
		t.Fatalf("alert count: %v", qerr)
	}
	if count != 0 {
		t.Fatalf("expected no alerts recorded for cancelled scan, got %d", count)
	}
}

// TestSchedulerScanOnceSucceedsNormally ensures the abort-on-cancel change
// doesn't break the happy path: with a live context and a matching log, the
// scan fires exactly one alert.
func TestSchedulerScanOnceSucceedsNormally(t *testing.T) {
	sched, rs, alertStore, logStore := setupSchedulerForContextTests(t, 2*time.Millisecond)
	seedMatchingRule(t, rs)
	if err := logStore.Store(context.Background(), model.NewLogEntry("svc-a", model.LevelError, "boom")); err != nil {
		t.Fatalf("seed log: %v", err)
	}

	if err := sched.ScanOnce(context.Background()); err != nil {
		t.Fatalf("ScanOnce: unexpected error: %v", err)
	}

	count, err := alertStore.Count(context.Background(), nil)
	if err != nil {
		t.Fatalf("alert count: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 alert recorded, got %d", count)
	}
}
