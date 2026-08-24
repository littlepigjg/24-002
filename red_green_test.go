package logalert

import (
	"context"
	"fmt"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/internal/service"
	"logalert/pkg/logger"
)

// TestDeferInLoopResourceAccumulation verifies that the defer-in-loop defect
// causes URL entries to be consumed during scheduler/cleanup operations.
// RED: entries are consumed (defect present)
// GREEN: entries are preserved (defect fixed)
func TestDeferInLoopResourceAccumulation(t *testing.T) {
	cfg := config.Default()
	log := logger.Default()

	ctx := context.Background()

	// Create stores
	logStore := store.NewMemoryLogStore(10000, log)
	alertStore := store.NewMemoryAlertStore(10000, log)
	ruleStore := store.NewMemoryRuleStore(log)
	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URLStore: %v", err)
	}

	// Create services
	ruleService := service.NewRuleService(ruleStore, cfg, log)
	alertService := service.NewAlertService(alertStore, cfg, log)
	scheduler := service.NewScheduler(ruleService, alertService, logStore, cfg, log)
	cleanup := service.NewCleanupService(logStore, alertStore, cfg, log)

	// Create URL service
	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URLService: %v", err)
	}

	// Inject URLStore into scheduler and cleanup
	scheduler.(interface{ SetURLStore(*store.URLStore) }).SetURLStore(urlStore)
	cleanup.(interface{ SetURLStore(*store.URLStore) }).SetURLStore(urlStore)

	// Create 10 short URLs
	numURLs := 10
	for i := 0; i < numURLs; i++ {
		req := &model.CreateReq{
			RawURL:     fmt.Sprintf("https://example.com/page-%d", i),
			CustomCode: fmt.Sprintf("short%02d", i),
		}
		_, err := urlService.Create(ctx, req)
		if err != nil {
			t.Fatalf("failed to create URL %d: %v", i, err)
		}
	}

	// Verify all URLs are created
	initialSnapshot := urlStore.RawSnapshot()
	if len(initialSnapshot) != numURLs {
		t.Fatalf("expected %d URLs created, got %d", numURLs, len(initialSnapshot))
	}
	t.Logf("Created %d URLs successfully", numURLs)

	// Run scheduler scan
	t.Log("Running scheduler ScanOnce...")
	if err := scheduler.ScanOnce(ctx); err != nil {
		t.Fatalf("ScanOnce failed: %v", err)
	}

	// Run cleanup
	t.Log("Running cleanup CleanupOnce...")
	if err := cleanup.CleanupOnce(ctx); err != nil {
		t.Fatalf("CleanupOnce failed: %v", err)
	}

	// Check final state
	finalSnapshot := urlStore.RawSnapshot()
	consumedCount := urlStore.ConsumedCount()
	pendingCount := urlStore.PendingCount()

	t.Logf("Final snapshot entries: %d", len(finalSnapshot))
	t.Logf("Consumed entries: %d", consumedCount)
	t.Logf("Pending tasks: %d", pendingCount)

	// The defect: defer-in-loop causes entries to be consumed prematurely.
	// With the defect: all entries are consumed (0 remaining in snapshot)
	// After fix: all entries should remain (numURLs in snapshot)
	if len(finalSnapshot) < numURLs {
		t.Logf("RED: defer-in-loop defect detected - %d entries consumed, %d remaining (expected %d remaining)",
			consumedCount, len(finalSnapshot), numURLs)
		t.Error("URL entries were consumed due to defer-in-loop resource accumulation defect")
	} else {
		t.Logf("GREEN: defer-in-loop defect fixed - all %d entries preserved", len(finalSnapshot))
	}

	// Additional verification: check specific consumption
	if consumedCount > 0 {
		t.Logf("Consumed count: %d - entries were prematurely consumed due to defer-in-loop", consumedCount)
	}
}

// TestDeferInLoopMultipleRuns verifies that the defect accumulates across multiple runs.
func TestDeferInLoopMultipleRuns(t *testing.T) {
	cfg := config.Default()
	log := logger.Default()

	ctx := context.Background()

	logStore := store.NewMemoryLogStore(10000, log)
	alertStore := store.NewMemoryAlertStore(10000, log)
	ruleStore := store.NewMemoryRuleStore(log)
	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URLStore: %v", err)
	}

	ruleService := service.NewRuleService(ruleStore, cfg, log)
	alertService := service.NewAlertService(alertStore, cfg, log)
	scheduler := service.NewScheduler(ruleService, alertService, logStore, cfg, log)
	cleanup := service.NewCleanupService(logStore, alertStore, cfg, log)
	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URLService: %v", err)
	}

	scheduler.(interface{ SetURLStore(*store.URLStore) }).SetURLStore(urlStore)
	cleanup.(interface{ SetURLStore(*store.URLStore) }).SetURLStore(urlStore)

	// Create 5 URLs initially
	numURLs := 5
	for i := 0; i < numURLs; i++ {
		req := &model.CreateReq{
			RawURL:     fmt.Sprintf("https://example.com/initial-%d", i),
			CustomCode: fmt.Sprintf("init%02d", i),
		}
		_, err := urlService.Create(ctx, req)
		if err != nil {
			t.Fatalf("failed to create URL: %v", err)
		}
	}

	// First run - should consume all 5 entries
	t.Log("=== First run ===")
	_ = scheduler.ScanOnce(ctx)
	_ = cleanup.CleanupOnce(ctx)
	snapshot1 := urlStore.RawSnapshot()
	consumed1 := urlStore.ConsumedCount()
	t.Logf("After first run: %d entries remain, %d consumed", len(snapshot1), consumed1)

	// Create 5 more URLs
	for i := 0; i < numURLs; i++ {
		req := &model.CreateReq{
			RawURL:     fmt.Sprintf("https://example.com/second-%d", i),
			CustomCode: fmt.Sprintf("second%02d", i),
		}
		_, err := urlService.Create(ctx, req)
		if err != nil {
			t.Fatalf("failed to create URL: %v", err)
		}
	}

	// Second run
	t.Log("=== Second run ===")
	_ = scheduler.ScanOnce(ctx)
	_ = cleanup.CleanupOnce(ctx)
	snapshot2 := urlStore.RawSnapshot()
	consumed2 := urlStore.ConsumedCount()
	t.Logf("After second run: %d entries remain, %d total consumed", len(snapshot2), consumed2)

	// The defect: entries get consumed during each scan cycle
	// After fix: entries should persist across multiple runs
	remaining := len(snapshot2)
	if remaining < numURLs {
		t.Logf("RED: defer-in-loop defect accumulates - only %d of %d expected entries remain",
			remaining, numURLs)
		t.Error("URL entries lost across multiple scan cycles due to defer-in-loop defect")
	} else {
		t.Logf("GREEN: all %d entries preserved across multiple runs", remaining)
	}
}
