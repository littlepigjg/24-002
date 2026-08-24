package store

import (
	"context"
	"errors"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// newTestLogStore builds a MemoryLogStore seeded with one entry for context tests.
func newTestLogStore(t *testing.T) *MemoryLogStore {
	t.Helper()
	s := NewMemoryLogStore(100, logger.Default())
	if err := s.Store(context.Background(), model.NewLogEntry("svc-a", model.LevelInfo, "hello")); err != nil {
		t.Fatalf("seed store: %v", err)
	}
	return s
}

// TestCountRespectsContextDeadline reproduces the reported bug: with a 100ms
// processing delay and a ~5ms context deadline, Count must return
// context.DeadlineExceeded instead of succeeding after the full delay.
func TestCountRespectsContextDeadline(t *testing.T) {
	s := newTestLogStore(t)
	s.SetProcessingDelay(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	// Let the deadline elapse so ctx is already done by the time we call.
	time.Sleep(10 * time.Millisecond)

	count, err := s.Count(ctx, nil)
	if err == nil {
		t.Fatalf("Count succeeded with %d entries; want context deadline exceeded", count)
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Count returned err=%v; want context.DeadlineExceeded", err)
	}
}

// TestQueryRespectsContextDeadline is the Query counterpart of the above.
func TestQueryRespectsContextDeadline(t *testing.T) {
	s := newTestLogStore(t)
	s.SetProcessingDelay(100 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()
	time.Sleep(10 * time.Millisecond)

	results, err := s.Query(ctx, nil, 10, 0)
	if err == nil {
		t.Fatalf("Query succeeded with %d results; want context deadline exceeded", len(results))
	}
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Query returned err=%v; want context.DeadlineExceeded", err)
	}
}

// TestCountSucceedsWhenContextNotExpired ensures the context-aware delay does
// not break the happy path when the deadline is comfortably in the future.
func TestCountSucceedsWhenContextNotExpired(t *testing.T) {
	s := newTestLogStore(t)
	s.SetProcessingDelay(5 * time.Millisecond)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	count, err := s.Count(ctx, nil)
	if err != nil {
		t.Fatalf("Count: unexpected error: %v", err)
	}
	if count != 1 {
		t.Fatalf("Count = %d; want 1", count)
	}
}

// TestCountHonoursCancelledContext ensures an already-cancelled context returns
// immediately without waiting out the processing delay.
func TestCountHonoursCancelledContext(t *testing.T) {
	s := newTestLogStore(t)
	s.SetProcessingDelay(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	start := time.Now()
	_, err := s.Count(ctx, nil)
	elapsed := time.Since(start)

	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Count returned err=%v; want context.Canceled", err)
	}
	if elapsed > 50*time.Millisecond {
		t.Fatalf("Count waited %v for a cancelled context; want to return promptly", elapsed)
	}
}
