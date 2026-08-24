package store

import (
	"context"
	"sync"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

func newTestAlertStore(t *testing.T, maxSize int) *MemoryAlertStore {
	t.Helper()
	return NewMemoryAlertStore(maxSize, logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))
}

func newTestAlert(i int) *model.AlertEvent {
	rule := &model.AlertRule{
		ID:       "rule-1",
		Name:     "test-rule",
		Severity: model.SeverityCritical,
		Window:   5 * time.Minute,
	}
	a := model.NewAlertEvent(rule, "test alert", "src")
	a.Severity = model.SeverityMedium // avoid noisy critical-existing warnings
	return a
}

// TestMemoryAlertStoreConcurrentRecord reproduces the load-test scenario: many
// goroutines recording alerts while readers query/snapshot the store. Before the
// fix, the unlocked map iteration in SnapshotAlerts/CountAlerts/GetAlertCount
// raced with writes and panicked with "concurrent map iteration and map write".
// Run with -race to verify the fix.
func TestMemoryAlertStoreConcurrentRecord(t *testing.T) {
	const (
		writers      = 30
		writesEach   = 200
		maxSize      = 10000
		readers      = 8
		stopAfterMs  = 500
	)

	store := newTestAlertStore(t, maxSize)
	store.SetAlertGuard(func(ctx context.Context, alert *model.AlertEvent) error {
		// Diagnostic hook: read a field to ensure no race on the alert itself.
		_ = alert.Severity
		return nil
	})

	ctx := context.Background()
	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Writers mirror the stress test: 30 goroutines, 200 writes each.
	wg.Add(writers)
	for w := 0; w < writers; w++ {
		go func() {
			defer wg.Done()
			for i := 0; i < writesEach; i++ {
				if err := store.Record(ctx, newTestAlert(i)); err != nil {
					t.Errorf("record failed: %v", err)
					return
				}
			}
		}()
	}

	// Readers hammer the unlocked diagnostic/metrics paths and query the list,
	// which is the "data missing/empty" symptom path.
	wg.Add(readers)
	for r := 0; r < readers; r++ {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				_ = store.CountAlerts()
				_ = store.GetAlertCount()
				_ = store.SnapshotAlerts()
				_, _ = store.ListRecent(ctx, 50)
				_, _ = store.Query(ctx, nil, 50, 0)
			}
		}()
	}

	// Concurrent status updates mutate stored events to exercise the same
	// *AlertEvent pointers the readers serialize.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
			}
			if ids := store.SnapshotAlerts(); len(ids) > 0 {
				_ = store.UpdateStatus(ctx, ids[0], model.AlertAcknowledged)
			}
		}
	}()

	time.Sleep(stopAfterMs * time.Millisecond)
	close(stop)
	wg.Wait()

	// Final consistency: total recorded (minus evictions) must not exceed max size.
	if got := store.CountAlerts(); got > maxSize {
		t.Fatalf("store exceeded max size: %d > %d", got, maxSize)
	}
	if got := store.CountAlerts(); got == 0 {
		t.Fatalf("expected alerts after concurrent writes, got 0")
	}
}
