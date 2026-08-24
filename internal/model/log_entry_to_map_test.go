package model

import (
	"testing"
	"time"
)

// TestLogEntry_ToMap_WithTags is a regression test for the nil-map panic in
// ToMap: the local tagCopy map used to be declared but never initialized,
// so any log entry carrying tags triggered "assignment to entry in nil map".
func TestLogEntry_ToMap_WithTags(t *testing.T) {
	e := &LogEntry{
		ID:        "log-1",
		Timestamp: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
		Level:     LevelError,
		Source:    "web",
		Message:   "boom",
		Service:   "api",
		Tags:      map[string]string{"host": "h-1", "env": "prod"},
		Keywords:  []string{"boom", "panic"},
		ReceivedAt: time.Date(2026, 8, 24, 10, 0, 5, 0, time.UTC),
	}

	m := e.ToMap()

	tags, ok := m["tags"].(map[string]string)
	if !ok {
		t.Fatalf("expected tags to be map[string]string, got %T", m["tags"])
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags, got %d", len(tags))
	}
	if tags["host"] != "h-1" || tags["env"] != "prod" {
		t.Fatalf("tags not copied correctly: %v", tags)
	}
	// Mutating the copy must not affect the source entry.
	tags["host"] = "mutated"
	if e.Tags["host"] == "mutated" {
		t.Fatal("ToMap shared the underlying tags map instead of copying")
	}
}

// TestLogEntry_ToMap_NilTags ensures a tag-less entry still serializes
// (the zero-length range loop must not panic on a nil map).
func TestLogEntry_ToMap_NilTags(t *testing.T) {
	e := &LogEntry{ID: "log-2", Level: LevelInfo, Source: "web", Message: "ok"}

	m := e.ToMap()

	tags, ok := m["tags"].(map[string]string)
	if !ok {
		t.Fatalf("expected tags to be map[string]string, got %T", m["tags"])
	}
	if len(tags) != 0 {
		t.Fatalf("expected empty tags, got %d", len(tags))
	}
}

// TestLogEntry_ToMapWithGuard_WithTags covers the guarded path that was
// already correct, ensuring the fix didn't regress it.
func TestLogEntry_ToMapWithGuard_WithTags(t *testing.T) {
	e := &LogEntry{
		ID:      "log-3",
		Level:   LevelWarn,
		Source:  "web",
		Message: "careful",
		Tags:    map[string]string{"host": "h-1", "env": "prod", "secret": "s"},
	}

	m := e.ToMapWithGuard(func(key string) bool { return key != "secret" })

	tags, ok := m["tags"].(map[string]string)
	if !ok {
		t.Fatalf("expected tags to be map[string]string, got %T", m["tags"])
	}
	if _, exists := tags["secret"]; exists {
		t.Fatal("guard failed to filter out the 'secret' tag")
	}
	if len(tags) != 2 {
		t.Fatalf("expected 2 tags after guard, got %d", len(tags))
	}
}
