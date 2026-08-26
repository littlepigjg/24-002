package service

import (
	"context"
	"errors"
	"strings"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// discardWriter is a LogWriter that drops all output, keeping tests quiet.
type discardWriter struct{}

func (discardWriter) Write(*logger.LogEntry) error { return nil }
func (discardWriter) Close() error                 { return nil }

func testLogger() logger.Logger {
	return logger.NewLogger(logger.LogLevelInfo, discardWriter{})
}

// newLogService builds a LogService backed by a fresh in-memory store.
func newLogService() (LogService, store.LogStore) {
	s := store.NewMemoryLogStore(100, testLogger())
	return NewLogService(s, &config.Config{}, testLogger()), s
}

// TestCreateLog_RejectsUnregisteredSource guards against the regression where
// a log entry carrying an unregistered source was silently accepted. The store
// returns a SOURCE_NOT_REGISTERED error, which the service must surface to the
// caller rather than swallow.
func TestCreateLog_RejectsUnregisteredSource(t *testing.T) {
	svc, st := newLogService()

	req := &model.CreateLogRequest{
		Level:   model.LevelError,
		Source:  "ghost-service",
		Message: "should be rejected",
	}

	entry, err := svc.CreateLog(context.Background(), req)
	if err == nil {
		t.Fatalf("expected error for unregistered source, got nil; entry=%+v", entry)
	}

	if !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}

	// The entry must not have been persisted.
	if entries, _ := st.Query(context.Background(), nil, 100, 0); len(entries) != 0 {
		t.Fatalf("expected zero stored entries, got %d", len(entries))
	}

	// A caller that later registers the source should then succeed, proving the
	// store registry (not the validation logic) is the gating mechanism.
	svc.RegisterSource("ghost-service")
	if _, err := svc.CreateLog(context.Background(), req); err != nil {
		t.Fatalf("expected success after registering source, got: %v", err)
	}
}

// TestCreateLog_AllowsRegisteredSource confirms the happy path still works once
// the source has been registered.
func TestCreateLog_AllowsRegisteredSource(t *testing.T) {
	svc, _ := newLogService()
	svc.RegisterSource("order-service")

	req := &model.CreateLogRequest{
		Level:   model.LevelInfo,
		Source:  "order-service",
		Message: "hello",
	}
	entry, err := svc.CreateLog(context.Background(), req)
	if err != nil {
		t.Fatalf("expected success for registered source, got: %v", err)
	}
	if entry == nil || entry.Source != "order-service" {
		t.Fatalf("unexpected entry: %+v", entry)
	}
}

// TestCreateLog_BatchRejectsUnregisteredSource mirrors the single-entry case for
// the batch path, which routes through the same store-error pipeline.
func TestCreateLog_BatchRejectsUnregisteredSource(t *testing.T) {
	svc, st := newLogService()

	reqs := []*model.CreateLogRequest{
		{Level: model.LevelInfo, Source: "ghost-service", Message: "m1"},
		{Level: model.LevelInfo, Source: "ghost-service", Message: "m2"},
	}

	entries, err := svc.CreateLogs(context.Background(), reqs)
	if err == nil {
		t.Fatalf("expected error for unregistered source in batch, got nil; entries=%d", len(entries))
	}
	if !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}
	if got, _ := st.Query(context.Background(), nil, 100, 0); len(got) != 0 {
		t.Fatalf("expected zero stored entries, got %d", len(got))
	}
}

// TestExtractStoreError_SurfacesStoreError pins the unit-level contract that was
// broken: errors.As must extract the concrete *store.StoreError instead of
// resetting it to nil.
func TestExtractStoreError_SurfacesStoreError(t *testing.T) {
	svc, _ := newLogService()
	ls := svc.(*logService)

	in := &store.StoreError{Code: "SOURCE_NOT_REGISTERED", Source: "x", Message: "boom"}
	out := ls.extractStoreError(in)
	if out == nil {
		t.Fatal("expected non-nil StoreError to be extracted, got nil")
	}
	if out.Code != "SOURCE_NOT_REGISTERED" {
		t.Fatalf("expected code %q, got %q", "SOURCE_NOT_REGISTERED", out.Code)
	}

	// A plain non-Store error must extract to nil, not crash.
	if got := ls.extractStoreError(errors.New("plain")); got != nil {
		t.Fatalf("expected nil for plain error, got %+v", got)
	}
}

// TestProcessStoreError_DoesNotSwallowPlainError pins that a generic store
// failure (not a *store.StoreError) is surfaced to the caller instead of being
// swallowed as success. The prior implementation returned nil for any error
// that did not unwrap to a StoreError, which is how unregistered sources (and
// other failures) leaked through as accepted data.
func TestProcessStoreError_DoesNotSwallowPlainError(t *testing.T) {
	svc, _ := newLogService()
	ls := svc.(*logService)

	plain := errors.New("store exploded")
	if err := ls.processStoreError(plain); err == nil {
		t.Fatal("expected plain store error to be surfaced, got nil")
	}

	// And the SOURCE_NOT_REGISTERED case must still map to the friendly message.
	se := &store.StoreError{Code: "SOURCE_NOT_REGISTERED", Source: "x", Message: "boom"}
	if err := ls.processStoreError(se); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}

	// nil stays nil.
	if err := ls.processStoreError(nil); err != nil {
		t.Fatalf("expected nil for nil input, got: %v", err)
	}
}

func newAlertService() (AlertService, store.AlertStore) {
	s := store.NewMemoryAlertStore(100, testLogger())
	return NewAlertService(s, &config.Config{}, testLogger()), s
}

// TestRecordAlert_RejectsUnregisteredSource guards the alert side of the same
// regression: an alert whose source is not registered must be rejected, not
// silently recorded.
func TestRecordAlert_RejectsUnregisteredSource(t *testing.T) {
	svc, st := newAlertService()

	rule := model.NewAlertRule("test", model.RuleCondition{
		Type:   model.ConditionCount,
		Source: "ghost-service",
	})
	alert := model.NewAlertEvent(rule, "triggered", "ghost-service")

	if err := svc.RecordAlert(context.Background(), alert); err == nil {
		t.Fatal("expected error for unregistered alert source, got nil")
	}

	if got, _ := st.Query(context.Background(), nil, 100, 0); len(got) != 0 {
		t.Fatalf("expected zero stored alerts, got %d", len(got))
	}

	// After registration the alert should be accepted.
	svc.RegisterSource("ghost-service")
	if err := svc.RecordAlert(context.Background(), alert); err != nil {
		t.Fatalf("expected success after registering alert source, got: %v", err)
	}
}

// TestExtractAlertStoreError_SurfacesStoreError pins the alert-side extractor.
func TestExtractAlertStoreError_SurfacesStoreError(t *testing.T) {
	svc, _ := newAlertService()
	as := svc.(*alertService)

	in := &store.StoreError{Code: "SOURCE_NOT_REGISTERED", Source: "x", Message: "boom"}
	out := as.extractAlertStoreError(in)
	if out == nil {
		t.Fatal("expected non-nil StoreError to be extracted, got nil")
	}
	if out.Code != "SOURCE_NOT_REGISTERED" {
		t.Fatalf("expected code %q, got %q", "SOURCE_NOT_REGISTERED", out.Code)
	}
	if got := as.extractAlertStoreError(errors.New("plain")); got != nil {
		t.Fatalf("expected nil for plain error, got %+v", got)
	}
}

// TestProcessAlertError_DoesNotSwallowPlainError pins the alert side of the
// same contract: plain store failures must be surfaced, not swallowed.
func TestProcessAlertError_DoesNotSwallowPlainError(t *testing.T) {
	svc, _ := newAlertService()
	as := svc.(*alertService)

	plain := errors.New("alert store exploded")
	if err := as.processAlertError(plain); err == nil {
		t.Fatal("expected plain alert error to be surfaced, got nil")
	}

	se := &store.StoreError{Code: "SOURCE_NOT_REGISTERED", Source: "x", Message: "boom"}
	if err := as.processAlertError(se); err == nil || !strings.Contains(err.Error(), "not registered") {
		t.Fatalf("expected 'not registered' error, got: %v", err)
	}

	if err := as.processAlertError(nil); err != nil {
		t.Fatalf("expected nil for nil input, got: %v", err)
	}
}
