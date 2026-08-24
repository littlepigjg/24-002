package service

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func newTestLogger() logger.Logger {
	// Use the discard writer so the reproduction stays quiet and side-effect free.
	return logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter())
}

func newTestConfig() *config.Config {
	return config.DefaultConfig()
}

// TestReproGetLogNotFound reproduces the reported issue:
// GetLog on a non-existent ID should be identifiable via errors.Is(err, ErrLogNotFound).
func TestReproGetLogNotFound(t *testing.T) {
	ctx := context.Background()
	logStore := store.NewMemoryLogStore(100, newTestLogger())
	svc := NewLogService(logStore, newTestConfig(), newTestLogger())

	_, err := svc.GetLog(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrLogNotFound) {
		t.Fatalf("expected errors.Is(err, ErrLogNotFound)=true, got false; err=%v", err)
	}
}

// TestReproDeleteLogNotFound reproduces the same issue for DeleteLog.
func TestReproDeleteLogNotFound(t *testing.T) {
	ctx := context.Background()
	logStore := store.NewMemoryLogStore(100, newTestLogger())
	svc := NewLogService(logStore, newTestConfig(), newTestLogger())

	err := svc.DeleteLog(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrLogNotFound) {
		t.Fatalf("expected errors.Is(err, ErrLogNotFound)=true, got false; err=%v", err)
	}
}

// TestReproAcknowledgeResolvedAlert reproduces the reported alert issue:
// acknowledging an already-resolved alert should be identifiable via errors.Is(err, ErrStateConflict).
func TestReproAcknowledgeResolvedAlert(t *testing.T) {
	ctx := context.Background()
	alertStore := store.NewMemoryAlertStore(100, newTestLogger())
	svc := NewAlertService(alertStore, newTestConfig(), newTestLogger())

	rule := &model.AlertRule{ID: "r1", Name: "rule-1", Severity: model.SeverityCritical}
	alert := model.NewAlertEvent(rule, "boom", "src")
	if err := alertStore.Record(ctx, alert); err != nil {
		t.Fatalf("setup Record failed: %v", err)
	}

	// Resolve it first.
	if _, err := svc.ResolveAlert(ctx, alert.ID); err != nil {
		t.Fatalf("setup ResolveAlert failed: %v", err)
	}

	// Now try to acknowledge the resolved alert -> expect ErrStateConflict.
	_, err := svc.AcknowledgeAlert(ctx, alert.ID, &model.AcknowledgeAlertRequest{User: "tester"})
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrStateConflict) {
		t.Fatalf("expected errors.Is(err, ErrStateConflict)=true, got false; err=%v", err)
	}
}

// TestReproGetAlertNotFound reproduces GetAlert not-found identification.
func TestReproGetAlertNotFound(t *testing.T) {
	ctx := context.Background()
	alertStore := store.NewMemoryAlertStore(100, newTestLogger())
	svc := NewAlertService(alertStore, newTestConfig(), newTestLogger())

	_, err := svc.GetAlert(ctx, "does-not-exist")
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
	if !errors.Is(err, ErrAlertNotFound) {
		t.Fatalf("expected errors.Is(err, ErrAlertNotFound)=true, got false; err=%v", err)
	}
}

// TestStorageFullStillDetected guards the case the user said already works,
// to make sure the fix doesn't regress capacity-exceeded detection.
func TestStorageFullStillDetected(t *testing.T) {
	ctx := context.Background()
	logStore := store.NewMemoryLogStore(1, newTestLogger())
	svc := NewLogService(logStore, newTestConfig(), newTestLogger())

	if _, err := svc.CreateLog(ctx, &model.CreateLogRequest{Source: "s", Level: model.LevelInfo, Message: "first"}); err != nil {
		t.Fatalf("first create failed: %v", err)
	}
	_, err := svc.CreateLog(ctx, &model.CreateLogRequest{Source: "s", Level: model.LevelInfo, Message: "second"})
	if err == nil {
		t.Fatal("expected storage-full error, got nil")
	}
	if !errors.Is(err, ErrStorageFull) {
		t.Fatalf("expected errors.Is(err, ErrStorageFull)=true, got false; err=%v", err)
	}
	fmt.Println("storage full detected OK")
}
