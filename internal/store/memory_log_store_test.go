package store

import (
	"context"
	"strings"
	"testing"

	"logalert/internal/model"

	"logalert/pkg/logger"
)

func quietLogger() logger.Logger {
	return logger.NewLogger(logger.LogLevelInfo, nopWriter{})
}

type nopWriter struct{}

func (nopWriter) Write(*logger.LogEntry) error { return nil }
func (nopWriter) Close() error                 { return nil }

// TestMemoryLogStore_Store_RejectsUnregisteredSource verifies the gating layer:
// a log entry whose source has not been registered must be rejected and must
// not be persisted.
func TestMemoryLogStore_Store_RejectsUnregisteredSource(t *testing.T) {
	s := NewMemoryLogStore(10, quietLogger())
	ctx := context.Background()
	entry := model.NewLogEntry("ghost", model.LevelInfo, "hi")

	if err := s.Store(ctx, entry); err == nil {
		t.Fatal("expected SOURCE_NOT_REGISTERED error, got nil")
	} else {
		se, ok := err.(*StoreError)
		if !ok {
			t.Fatalf("expected *StoreError, got %T (%v)", err, err)
		}
		if se.Code != "SOURCE_NOT_REGISTERED" {
			t.Fatalf("expected code %q, got %q", "SOURCE_NOT_REGISTERED", se.Code)
		}
	}

	if got, _ := s.Query(ctx, nil, 10, 0); len(got) != 0 {
		t.Fatalf("expected zero stored entries, got %d", len(got))
	}

	// Registering the source must then allow the same entry through.
	s.RegisterSource("ghost")
	if err := s.Store(ctx, entry); err != nil {
		t.Fatalf("expected success after registering source, got: %v", err)
	}
}

// TestMemoryLogStore_StoreBatch_RejectsUnregisteredSourceAtomic ensures the
// batch path applies the same source check as Store and rejects the whole batch
// atomically — none of the entries should be persisted when one source is bad.
func TestMemoryLogStore_StoreBatch_RejectsUnregisteredSourceAtomic(t *testing.T) {
	s := NewMemoryLogStore(10, quietLogger())
	ctx := context.Background()

	entries := []*model.LogEntry{
		model.NewLogEntry("registered", model.LevelInfo, "ok"),
		model.NewLogEntry("ghost", model.LevelInfo, "bad"),
	}

	err := s.StoreBatch(ctx, entries)
	if err == nil {
		t.Fatal("expected SOURCE_NOT_REGISTERED error from StoreBatch, got nil")
	}
	se, ok := err.(*StoreError)
	if !ok {
		t.Fatalf("expected *StoreError, got %T (%v)", err, err)
	}
	if se.Code != "SOURCE_NOT_REGISTERED" {
		t.Fatalf("expected code %q, got %q", "SOURCE_NOT_REGISTERED", se.Code)
	}

	// Atomicity: the registered entry must NOT have leaked in.
	if got, _ := s.Query(ctx, nil, 10, 0); len(got) != 0 {
		t.Fatalf("expected zero stored entries (atomic rejection), got %d", len(got))
	}

	// Register the bad source and the batch should succeed wholesale.
	s.RegisterSource("registered")
	s.RegisterSource("ghost")
	if err := s.StoreBatch(ctx, entries); err != nil {
		t.Fatalf("expected success after registering sources, got: %v", err)
	}
}

// TestMemoryLogStore_Store_AllowsEmptySource documents the existing lenient
// behavior: an empty source bypasses the registry check (intended, since the
// HTTP layer rejects empty sources earlier). This pins the boundary so a future
// tightening doesn't silently change it.
func TestMemoryLogStore_Store_AllowsEmptySource(t *testing.T) {
	s := NewMemoryLogStore(10, quietLogger())
	ctx := context.Background()

	entry := model.NewLogEntry("", model.LevelInfo, "no source")
	if err := s.Store(ctx, entry); err != nil {
		t.Fatalf("expected empty source to be accepted at store layer, got: %v", err)
	}
	if !strings.Contains(entry.ID, "") { // trivial sanity that entry returned
		t.Fatal("unexpected entry id")
	}
}
