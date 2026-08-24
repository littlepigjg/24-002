package service

import (
	"context"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// noopLogStore satisfies store.LogStore for serialization-path tests; only
// Store is exercised, the rest return zero values.
type noopLogStore struct{}

func (noopLogStore) Store(context.Context, *model.LogEntry) error                  { return nil }
func (noopLogStore) StoreBatch(context.Context, []*model.LogEntry) error           { return nil }
func (noopLogStore) Get(context.Context, string) (*model.LogEntry, error)         { return nil, nil }
func (noopLogStore) Query(context.Context, *model.LogFilter, int, int) ([]*model.LogEntry, error) {
	return nil, nil
}
func (noopLogStore) Count(context.Context, *model.LogFilter) (int64, error) { return 0, nil }
func (noopLogStore) Delete(context.Context, string) error                   { return nil }
func (noopLogStore) DeleteExpired(context.Context, time.Time) (int64, error) { return 0, nil }
func (noopLogStore) ListSources(context.Context) ([]string, error)          { return nil, nil }
func (noopLogStore) ListServices(context.Context) ([]string, error)         { return nil, nil }
func (noopLogStore) Statistics(context.Context, time.Time, time.Time) (*store.LogStatistics, error) {
	return nil, nil
}
func (noopLogStore) HourlyBreakdown(context.Context, time.Time, time.Time) ([]store.HourlyCount, error) {
	return nil, nil
}
func (noopLogStore) Close() error { return nil }

// noopAlertStore satisfies store.AlertStore; only Record is exercised.
type noopAlertStore struct{}

func (noopAlertStore) Record(context.Context, *model.AlertEvent) error { return nil }
func (noopAlertStore) Get(context.Context, string) (*model.AlertEvent, error) {
	return nil, nil
}
func (noopAlertStore) UpdateStatus(context.Context, string, model.AlertStatus) error { return nil }
func (noopAlertStore) Query(context.Context, *model.AlertFilter, int, int) ([]*model.AlertEvent, error) {
	return nil, nil
}
func (noopAlertStore) Count(context.Context, *model.AlertFilter) (int64, error) { return 0, nil }
func (noopAlertStore) GetByRule(context.Context, string, int, int) ([]*model.AlertEvent, error) {
	return nil, nil
}
func (noopAlertStore) GetByStatus(context.Context, model.AlertStatus, int, int) ([]*model.AlertEvent, error) {
	return nil, nil
}
func (noopAlertStore) ListRecent(context.Context, int) ([]*model.AlertEvent, error) { return nil, nil }
func (noopAlertStore) Delete(context.Context, string) error                       { return nil }
func (noopAlertStore) DeleteOld(context.Context, time.Time) (int64, error)         { return 0, nil }
func (noopAlertStore) Close() error                                                { return nil }

// TestCreateLog_SerializeTaggedEntry is a regression test for the nil-map
// panic in logService.serializeEntryForLog: a tagged log entry used to crash
// "assignment to entry in nil map" when the service serialized it for logging.
func TestCreateLog_SerializeTaggedEntry(t *testing.T) {
	svc := NewLogService(noopLogStore{}, config.DefaultConfig(), logger.ConsoleLogger(logger.LogLevelDebug))

	req := &model.CreateLogRequest{
		Level:   model.LevelError,
		Source:  "web",
		Message: "boom",
		Service: "api",
		Tags:    map[string]string{"host": "h-1", "env": "prod"},
	}

	entry, err := svc.CreateLog(context.Background(), req)
	if err != nil {
		t.Fatalf("CreateLog failed: %v", err)
	}
	if entry == nil {
		t.Fatal("expected non-nil entry")
	}
}

// TestRecordAlert_SerializeDetailedAlert is a regression test for the nil-map
// panic in alertService.serializeAlertForLog: an alert with details used to
// crash "assignment to entry in nil map" when the service serialized it.
func TestRecordAlert_SerializeDetailedAlert(t *testing.T) {
	svc := NewAlertService(noopAlertStore{}, config.DefaultConfig(), logger.ConsoleLogger(logger.LogLevelDebug))

	alert := &model.AlertEvent{
		ID:          "alert-1",
		RuleID:      "rule-1",
		RuleName:    "high error rate",
		Severity:    model.SeverityCritical,
		Status:      model.AlertOpen,
		Message:     "error rate above threshold",
		Source:      "web",
		Service:     "api",
		Details:     map[string]interface{}{"count": 42, "window": "5m"},
		TriggeredAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}

	if err := svc.RecordAlert(context.Background(), alert); err != nil {
		t.Fatalf("RecordAlert failed: %v", err)
	}
}
