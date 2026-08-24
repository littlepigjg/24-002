package main

import (
	"fmt"
	"testing"
	"time"

	"logalert/internal/model"
)

func TestRedGreen(t *testing.T) {
	allPassed := true

	t.Log("=== Test 1: LogEntry.ToMap() with Tags ===")
	if err := safeTest(testLogEntryToMap); err != nil {
		t.Logf("FAIL: %v", err)
		allPassed = false
	} else {
		t.Log("PASS")
	}

	t.Log("=== Test 2: AlertEvent.ToMap() with Details ===")
	if err := safeTest(testAlertEventToMap); err != nil {
		t.Logf("FAIL: %v", err)
		allPassed = false
	} else {
		t.Log("PASS")
	}

	t.Log("=== Test 3: LogEntry.ToMapWithGuard() ===")
	if err := safeTest(testLogEntryToMapWithGuard); err != nil {
		t.Logf("FAIL: %v", err)
		allPassed = false
	} else {
		t.Log("PASS")
	}

	t.Log("=== Test 4: AlertEvent.ToMapWithGuard() ===")
	if err := safeTest(testAlertEventToMapWithGuard); err != nil {
		t.Logf("FAIL: %v", err)
		allPassed = false
	} else {
		t.Log("PASS")
	}

	if allPassed {
		t.Log("GREEN（绿灯，缺陷已修复）")
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	} else {
		t.Log("RED（红灯，缺陷未修复）")
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatal("缺陷未修复：ToMap 方法向未初始化的 map 写入导致 panic")
	}
}

func safeTest(fn func() error) (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("panic: %v", r)
		}
	}()
	return fn()
}

func testLogEntryToMap() error {
	entry := model.NewLogEntry("test-service", model.LevelInfo, "test message")
	entry.Service = "api-gateway"
	entry.Tags["host"] = "server-01"
	entry.Tags["env"] = "production"
	entry.Tags["version"] = "v2.1.0"
	entry.Keywords = []string{"error", "timeout"}

	result := entry.ToMap()

	if result == nil {
		return fmt.Errorf("ToMap returned nil")
	}
	if result["id"] != entry.ID {
		return fmt.Errorf("id mismatch: got %v, want %v", result["id"], entry.ID)
	}
	if result["level"] != model.LevelInfo {
		return fmt.Errorf("level mismatch: got %v (type %T), want %v (type %T)", result["level"], result["level"], model.LevelInfo, model.LevelInfo)
	}
	if result["source"] != "test-service" {
		return fmt.Errorf("source mismatch: got %v, want test-service", result["source"])
	}
	if result["message"] != "test message" {
		return fmt.Errorf("message mismatch: got %v, want 'test message'", result["message"])
	}
	if result["service"] != "api-gateway" {
		return fmt.Errorf("service mismatch: got %v, want api-gateway", result["service"])
	}

	tagsRaw, ok := result["tags"].(map[string]string)
	if !ok {
		return fmt.Errorf("tags is not map[string]string, got %T", result["tags"])
	}
	if len(tagsRaw) != 3 {
		return fmt.Errorf("tags count mismatch: got %d, want 3", len(tagsRaw))
	}
	if tagsRaw["host"] != "server-01" {
		return fmt.Errorf("tags[host] mismatch: got %v, want server-01", tagsRaw["host"])
	}
	if tagsRaw["env"] != "production" {
		return fmt.Errorf("tags[env] mismatch: got %v, want production", tagsRaw["env"])
	}
	if tagsRaw["version"] != "v2.1.0" {
		return fmt.Errorf("tags[version] mismatch: got %v, want v2.1.0", tagsRaw["version"])
	}

	kw, ok := result["keywords"].([]string)
	if !ok {
		return fmt.Errorf("keywords is not []string, got %T", result["keywords"])
	}
	if len(kw) != 2 {
		return fmt.Errorf("keywords count mismatch: got %d, want 2", len(kw))
	}

	return nil
}

func testAlertEventToMap() error {
	rule := model.NewAlertRule("high-error-rate", model.RuleCondition{
		Type:   model.ConditionErrorRate,
		Level:  model.LevelError,
		Source: "api-gateway",
	})
	alert := model.NewAlertEvent(rule, "Error rate exceeded threshold", "api-gateway")
	alert.Details["count"] = 42
	alert.Details["window"] = "5m"
	alert.Details["threshold"] = 0.85
	alert.Details["service"] = "checkout"

	result := alert.ToMap()

	if result == nil {
		return fmt.Errorf("ToMap returned nil")
	}
	if result["id"] != alert.ID {
		return fmt.Errorf("id mismatch: got %v, want %v", result["id"], alert.ID)
	}
	if result["rule_id"] != rule.ID {
		return fmt.Errorf("rule_id mismatch: got %v, want %v", result["rule_id"], rule.ID)
	}
	if result["rule_name"] != "high-error-rate" {
		return fmt.Errorf("rule_name mismatch: got %v, want high-error-rate", result["rule_name"])
	}
	if result["severity"] != model.SeverityMedium {
		return fmt.Errorf("severity mismatch: got %v (type %T), want %v (type %T)", result["severity"], result["severity"], model.SeverityMedium, model.SeverityMedium)
	}
	if result["status"] != model.AlertOpen {
		return fmt.Errorf("status mismatch: got %v (type %T), want %v (type %T)", result["status"], result["status"], model.AlertOpen, model.AlertOpen)
	}
	if result["message"] != "Error rate exceeded threshold" {
		return fmt.Errorf("message mismatch: got %v", result["message"])
	}
	if result["source"] != "api-gateway" {
		return fmt.Errorf("source mismatch: got %v, want api-gateway", result["source"])
	}

	detailsRaw, ok := result["details"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("details is not map[string]interface{}, got %T", result["details"])
	}
	if len(detailsRaw) != 4 {
		return fmt.Errorf("details count mismatch: got %d, want 4", len(detailsRaw))
	}
	if detailsRaw["count"] != 42 {
		return fmt.Errorf("details[count] mismatch: got %v, want 42", detailsRaw["count"])
	}
	if detailsRaw["window"] != "5m" {
		return fmt.Errorf("details[window] mismatch: got %v, want 5m", detailsRaw["window"])
	}

	return nil
}

func testLogEntryToMapWithGuard() error {
	entry := model.NewLogEntry("auth-service", model.LevelWarn, "auth token expired")
	entry.Tags["host"] = "auth-01"
	entry.Tags["env"] = "staging"

	guard := func(key string) bool {
		return key != "secret"
	}

	result := entry.ToMapWithGuard(guard)

	if result == nil {
		return fmt.Errorf("ToMapWithGuard returned nil")
	}

	tagsRaw, ok := result["tags"].(map[string]string)
	if !ok {
		return fmt.Errorf("tags is not map[string]string, got %T", result["tags"])
	}
	if len(tagsRaw) != 2 {
		return fmt.Errorf("filtered tags count mismatch: got %d, want 2", len(tagsRaw))
	}
	if tagsRaw["host"] != "auth-01" {
		return fmt.Errorf("tags[host] mismatch: got %v, want auth-01", tagsRaw["host"])
	}

	allResult := entry.ToMapWithGuard(nil)
	allTags, ok := allResult["tags"].(map[string]string)
	if !ok {
		return fmt.Errorf("full tags is not map[string]string")
	}
	if len(allTags) != 2 {
		return fmt.Errorf("unfiltered tags count mismatch: got %d, want 2", len(allTags))
	}

	return nil
}

func testAlertEventToMapWithGuard() error {
	rule := model.NewAlertRule("latency-spike", model.RuleCondition{
		Type:   model.ConditionCount,
		Level:  model.LevelError,
		Source: "payment-svc",
	})
	alert := model.NewAlertEvent(rule, "Latency spike detected", "payment-svc")
	alert.Details["p99"] = 2500
	alert.Details["region"] = "us-east-1"

	guard := func(key string) bool {
		return key != "secret_key"
	}

	result := alert.ToMapWithGuard(guard)

	if result == nil {
		return fmt.Errorf("ToMapWithGuard returned nil")
	}

	detailsRaw, ok := result["details"].(map[string]interface{})
	if !ok {
		return fmt.Errorf("details is not map[string]interface{}, got %T", result["details"])
	}
	if len(detailsRaw) != 2 {
		return fmt.Errorf("filtered details count mismatch: got %d, want 2", len(detailsRaw))
	}

	return nil
}

func init() {
	time.Local = time.UTC
}
