package model

import (
	"testing"
	"time"
)

// TestAlertEvent_ToMap_WithDetails is a regression test for the nil-map panic
// in ToMap: the local detailCopy map used to be declared but never
// initialized, so any alert carrying details triggered
// "assignment to entry in nil map".
func TestAlertEvent_ToMap_WithDetails(t *testing.T) {
	a := &AlertEvent{
		ID:          "alert-1",
		RuleID:      "rule-1",
		RuleName:    "high error rate",
		Severity:    SeverityCritical,
		Status:      AlertOpen,
		Message:     "error rate above threshold",
		Source:      "web",
		Service:     "api",
		Details:     map[string]interface{}{"count": 42, "window": "5m"},
		TriggeredAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}

	m := a.ToMap()

	details, ok := m["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected details to be map[string]interface{}, got %T", m["details"])
	}
	if len(details) != 2 {
		t.Fatalf("expected 2 details, got %d", len(details))
	}
	if count, _ := details["count"].(int); count != 42 {
		t.Fatalf("expected count=42, got %v", details["count"])
	}
	// Mutating the copy must not affect the source alert.
	details["count"] = 0
	if a.Details["count"] == 0 {
		t.Fatal("ToMap shared the underlying details map instead of copying")
	}
}

// TestAlertEvent_ToMap_NilDetails ensures a detail-less alert still serializes.
func TestAlertEvent_ToMap_NilDetails(t *testing.T) {
	a := &AlertEvent{
		ID:          "alert-2",
		Severity:    SeverityHigh,
		Status:      AlertOpen,
		Source:      "web",
		TriggeredAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}

	m := a.ToMap()

	details, ok := m["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected details to be map[string]interface{}, got %T", m["details"])
	}
	if len(details) != 0 {
		t.Fatalf("expected empty details, got %d", len(details))
	}
}

// TestAlertEvent_ToMapWithGuard_WithDetails covers the already-correct guarded
// path to guard against regressions.
func TestAlertEvent_ToMapWithGuard_WithDetails(t *testing.T) {
	a := &AlertEvent{
		ID:          "alert-3",
		Severity:    SeverityCritical,
		Status:      AlertOpen,
		Source:      "web",
		Details:     map[string]interface{}{"count": 7, "window": "5m", "secret": "s"},
		TriggeredAt: time.Date(2026, 8, 24, 10, 0, 0, 0, time.UTC),
	}

	m := a.ToMapWithGuard(func(key string) bool { return key != "secret" })

	details, ok := m["details"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected details to be map[string]interface{}, got %T", m["details"])
	}
	if _, exists := details["secret"]; exists {
		t.Fatal("guard failed to filter out the 'secret' detail")
	}
	if len(details) != 2 {
		t.Fatalf("expected 2 details after guard, got %d", len(details))
	}
}
