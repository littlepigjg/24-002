package retry_test

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"logalert/pkg/cache"
	"logalert/pkg/errors"
	"logalert/pkg/retry"
)

// countLoader returns a not_found error on every call and records how many
// times it was invoked. This is the permanent error the bug report is about.
func newNotFoundLoader(calls *int32) func() (interface{}, error) {
	return func() (interface{}, error) {
		atomic.AddInt32(calls, 1)
		return nil, errors.NotFoundErrorFactory.New(404, "resource not found")
	}
}

func newDatabaseLoader(calls *int32) func() (interface{}, error) {
	return func() (interface{}, error) {
		atomic.AddInt32(calls, 1)
		return nil, errors.DatabaseErrorFactory.New(500, "transient db error")
	}
}

// loadThroughCache wraps the loader in cache.GetOrSet, exactly as the bug
// report describes, then runs it under retry.Do with the default config.
func loadThroughCache(t *testing.T, lc *cache.LazyCache, loader func() (interface{}, error), calls *int32) error {
	t.Helper()
	return retry.Do(context.Background(), retry.DefaultConfig(), func() error {
		_, err := lc.GetOrSet("k", loader, time.Minute)
		return err
	})
}

// Test_RetryCachePermanentErrorNotRetied reproduces the reported bug: a
// not_found error returned by the cache's loader must NOT be retried, because
// not_found is a permanent error. Before the fix the loader ran 3 times.
func Test_RetryCachePermanentErrorNotRetied(t *testing.T) {
	var calls int32
	lc := cache.NewLazy(100)

	err := loadThroughCache(t, lc, newNotFoundLoader(&calls), &calls)

	if got, want := atomic.LoadInt32(&calls), int32(1); got != want {
		t.Fatalf("permanent (not_found) loader should run exactly once, ran %d times", got)
	}
	if err == nil {
		t.Fatal("expected the not_found error to propagate, got nil")
	}
	// The original not_found DetailedError must survive the cache wrapping.
	if et, ok := errors.TypeOfWrapped(err); !ok || et != errors.ErrorTypeNotFound {
		t.Fatalf("propagated error should unwrap to not_found, got type=%q ok=%v: %v", et, ok, err)
	}
}

// Test_RetryCacheTransientErrorIsRetied guards the inverse: a database error
// is transient and MUST still be retried MaxAttempts times. This ensures the
// fix did not silently disable retry for genuinely transient errors.
func Test_RetryCacheTransientErrorIsRetied(t *testing.T) {
	var calls int32
	lc := cache.NewLazy(100)

	_ = loadThroughCache(t, lc, newDatabaseLoader(&calls), &calls)

	if got, want := atomic.LoadInt32(&calls), int32(retry.DefaultConfig().MaxAttempts); got != want {
		t.Fatalf("transient (database) loader should be retried %d times, ran %d times", want, got)
	}
}

// Test_RetryDirectPermanentErrorBaseline is the "direct, no cache" baseline
// from the bug report: passing not_found straight to retry.Do must return on
// the first attempt. This documents the behaviour the cache path must match.
func Test_RetryDirectPermanentErrorBaseline(t *testing.T) {
	var calls int32
	ctx := context.Background()
	cfg := retry.DefaultConfig()
	err := retry.Do(ctx, cfg, func() error {
		atomic.AddInt32(&calls, 1)
		return errors.NotFoundErrorFactory.New(404, "resource not found")
	})
	if got, want := atomic.LoadInt32(&calls), int32(1); got != want {
		t.Fatalf("direct not_found should run once, ran %d times", got)
	}
	if err == nil {
		t.Fatal("expected the not_found error to propagate, got nil")
	}
}
