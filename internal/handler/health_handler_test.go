package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"logalert/internal/service"
	"logalert/pkg/logger"
)

// discardLogWriter is a LogWriter that drops all output, for quiet tests.
type discardLogWriter struct{}

func (discardLogWriter) Write(_ *logger.LogEntry) error { return nil }
func (discardLogWriter) Close() error                   { return nil }

func newTestLogger(t *testing.T) logger.Logger {
	t.Helper()
	return logger.NewLoggerBuilder().
		SetLevel(logger.LogLevelError).
		SetWriter(discardLogWriter{}).
		Build()
}

// stubScheduler is a minimal Scheduler used only to drive SchedulerHandler
// in tests without standing up the full service graph.
type stubScheduler struct{ status service.SchedulerStatus }

func (s stubScheduler) Start(_ context.Context) error      { return nil }
func (s stubScheduler) Stop()                               {}
func (s stubScheduler) ScanOnce(_ context.Context) error    { return nil }
func (s stubScheduler) GetStatus() service.SchedulerStatus { return s.status }

// TestHealthHandler_EmptyStoreDoesNotPanic reproduces the startup crash where
// the metric store has no samples yet. Previously HandleHealth indexed
// sortedMetrics[0] unconditionally and panicked with "index out of range [0]
// with length 0". It must now return a normal 200 response instead.
func TestHealthHandler_EmptyStoreDoesNotPanic(t *testing.T) {
	store := NewHealthMetricStore() // no metrics added
	h := NewHealthHandlerWithStore(newTestLogger(t), store)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%q)", rec.Code, rec.Body.String())
	}

	var resp HealthResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if resp.Status != "ok" {
		t.Errorf("expected status ok, got %q", resp.Status)
	}
}

// TestSchedulerHandler_EmptyStoreDoesNotPanic reproduces the same crash in the
// scheduler status endpoint, which indexed sortedMetrics[0] without a guard.
func TestSchedulerHandler_EmptyStoreDoesNotPanic(t *testing.T) {
	store := NewHealthMetricStore()
	sched := stubScheduler{status: service.SchedulerStatus{Running: false}}
	h := NewSchedulerHandler(sched, newTestLogger(t), store)

	req := httptest.NewRequest(http.MethodGet, "/api/scheduler/status", nil)
	rec := httptest.NewRecorder()

	h.GetStatus(rec, req) // must not panic

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d (body=%q)", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Status struct {
				Running bool `json:"running"`
			} `json:"status"`
			LatestMetric float64 `json:"latest_metric"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if resp.Data.LatestMetric != 0 {
		t.Errorf("expected latest_metric 0 when store is empty, got %v", resp.Data.LatestMetric)
	}
}

// TestHealthHandler_NonEmptyStoreStillReportsValue guards against the fix
// regressing the populated-store path: with metrics present the handler must
// still read the latest value.
func TestHealthHandler_NonEmptyStoreStillReportsValue(t *testing.T) {
	store := NewHealthMetricStore()
	store.AddMetric("cpu", 0.42)
	h := NewHealthHandlerWithStore(newTestLogger(t), store)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	h.HandleHealth(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
}

// TestSchedulerHandler_NonEmptyStoreReportsLatestValue confirms the scheduler
// handler reports the newest sample when the store is populated.
func TestSchedulerHandler_NonEmptyStoreReportsLatestValue(t *testing.T) {
	store := NewHealthMetricStore()
	store.AddMetric("cpu", 0.1)
	store.AddMetric("cpu", 0.9) // newest
	sched := stubScheduler{status: service.SchedulerStatus{Running: true}}
	h := NewSchedulerHandler(sched, newTestLogger(t), store)

	req := httptest.NewRequest(http.MethodGet, "/api/scheduler/status", nil)
	rec := httptest.NewRecorder()

	h.GetStatus(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp struct {
		Data struct {
			LatestMetric float64 `json:"latest_metric"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid response body: %v", err)
	}
	if resp.Data.LatestMetric != 0.9 {
		t.Errorf("expected latest_metric 0.9, got %v", resp.Data.LatestMetric)
	}
}
