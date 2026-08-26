package service

import (
	"context"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// discardWriter is a no-op LogWriter for quiet tests.
type discardWriter struct{}

func (discardWriter) Write(*logger.LogEntry) error { return nil }
func (discardWriter) Close() error                 { return nil }

func newTestLogger() logger.Logger {
	return logger.NewLogger(logger.LogLevelFatal, discardWriter{})
}

// TestGetErrorRateTrend_WithAlerts reproduces the reported scenario:
// 20 logs (10 ERROR / 10 INFO) and 10 alerts (3 high / 7 low). The correct
// error rate per hour is 0.725. A slice-aliasing bug in the alert-adjustment
// block previously inflated this to 1.5.
func TestGetErrorRateTrend_WithAlerts(t *testing.T) {
	now := time.Now().Truncate(time.Second)

	logStore := store.NewMemoryLogStore(1000, newTestLogger())
	alertStore := store.NewMemoryAlertStore(1000, newTestLogger())

	// 10 ERROR + 10 INFO logs, all in the current hour.
	for i := 0; i < 10; i++ {
		entry := model.NewLogEntry("svc", model.LevelError, "boom")
		entry.Timestamp = now
		if err := logStore.Store(context.Background(), entry); err != nil {
			t.Fatalf("store error log %d: %v", i, err)
		}
	}
	for i := 0; i < 10; i++ {
		entry := model.NewLogEntry("svc", model.LevelInfo, "ok")
		entry.Timestamp = now
		if err := logStore.Store(context.Background(), entry); err != nil {
			t.Fatalf("store info log %d: %v", i, err)
		}
	}

	// 3 high + 7 low severity alerts.
	highRule := model.NewAlertRule("high-rule", model.RuleCondition{})
	highRule.Severity = model.SeverityHigh
	lowRule := model.NewAlertRule("low-rule", model.RuleCondition{})
	lowRule.Severity = model.SeverityLow
	for i := 0; i < 3; i++ {
		evt := model.NewAlertEvent(highRule, "high alert", "svc")
		evt.TriggeredAt = now
		if err := alertStore.Record(context.Background(), evt); err != nil {
			t.Fatalf("record high alert %d: %v", i, err)
		}
	}
	for i := 0; i < 7; i++ {
		evt := model.NewAlertEvent(lowRule, "low alert", "svc")
		evt.TriggeredAt = now
		if err := alertStore.Record(context.Background(), evt); err != nil {
			t.Fatalf("record low alert %d: %v", i, err)
		}
	}

	svc := NewStatsService(logStore, alertStore, nil, newTestLogger())

	req := &model.StatsRequest{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
		GroupBy:   "hour",
	}

	points, err := svc.GetErrorRateTrend(context.Background(), req)
	if err != nil {
		t.Fatalf("GetErrorRateTrend: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected at least one data point")
	}

	var pt ErrorRatePoint
	for _, p := range points {
		pt = p
	}

	// Counts must be unaffected by the adjustment factor.
	if pt.TotalCount != 20 {
		t.Errorf("TotalCount = %d, want 20", pt.TotalCount)
	}
	if pt.ErrorCount != 10 {
		t.Errorf("ErrorCount = %d, want 10", pt.ErrorCount)
	}

	// Expected: base 0.5 * (1 + alertErrorRatio*(1+logMatchRatio))
	// = 0.5 * (1 + (3/10)*(1 + 10/20)) = 0.5 * (1 + 0.3*1.5) = 0.725.
	const wantRate = 0.725
	if !approxEqual(pt.ErrorRate, wantRate, 1e-9) {
		t.Errorf("ErrorRate = %.4f, want %.4f", pt.ErrorRate, wantRate)
	}
}

// TestGetErrorRateTrend_NoAlerts confirms the adjustment factor is skipped
// (alertStore nil) and the rate is the plain base error rate.
func TestGetErrorRateTrend_NoAlerts(t *testing.T) {
	now := time.Now().Truncate(time.Second)
	logStore := store.NewMemoryLogStore(1000, newTestLogger())

	for i := 0; i < 10; i++ {
		entry := model.NewLogEntry("svc", model.LevelError, "boom")
		entry.Timestamp = now
		_ = logStore.Store(context.Background(), entry)
	}
	for i := 0; i < 10; i++ {
		entry := model.NewLogEntry("svc", model.LevelInfo, "ok")
		entry.Timestamp = now
		_ = logStore.Store(context.Background(), entry)
	}

	svc := NewStatsService(logStore, nil, nil, newTestLogger())
	req := &model.StatsRequest{
		StartTime: now.Add(-time.Hour),
		EndTime:   now.Add(time.Hour),
		GroupBy:   "hour",
	}

	points, err := svc.GetErrorRateTrend(context.Background(), req)
	if err != nil {
		t.Fatalf("GetErrorRateTrend: %v", err)
	}
	if len(points) == 0 {
		t.Fatal("expected at least one data point")
	}

	var pt ErrorRatePoint
	for _, p := range points {
		pt = p
	}
	if !approxEqual(pt.ErrorRate, 0.5, 1e-9) {
		t.Errorf("ErrorRate = %.4f, want 0.5", pt.ErrorRate)
	}
}

func approxEqual(a, b, eps float64) bool {
	d := a - b
	if d < 0 {
		d = -d
	}
	return d <= eps
}
