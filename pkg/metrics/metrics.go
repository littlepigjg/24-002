// Package metrics provides basic metrics collection for the application.
package metrics

import (
	"sync"
	"sync/atomic"
	"time"
)

// Metrics holds application-level metrics.
type Metrics struct {
	mu sync.RWMutex

	// Request counters
	TotalRequests    atomic.Int64
	SuccessfulRequests atomic.Int64
	FailedRequests    atomic.Int64

	// Log counters
	LogsCreated    atomic.Int64
	LogsQueried    atomic.Int64
	LogsDeleted    atomic.Int64

	// Alert counters
	AlertsCreated    atomic.Int64
	AlertsAcknowledged atomic.Int64
	AlertsResolved   atomic.Int64

	// Rule counters
	RulesCreated atomic.Int64
	RulesUpdated atomic.Int64
	RulesDeleted atomic.Int64

	// Scheduler
	SchedulerScans   atomic.Int64
	SchedulerErrors  atomic.Int64

	// Timing
	StartTime time.Time

	// Custom metrics
	customMetrics map[string]*atomic.Int64
}

// Global metrics instance
var globalMetrics *Metrics
var globalOnce sync.Once

// Global returns the global Metrics instance.
func Global() *Metrics {
	globalOnce.Do(func() {
		globalMetrics = NewMetrics()
	})
	return globalMetrics
}

// NewMetrics creates a new Metrics instance.
func NewMetrics() *Metrics {
	return &Metrics{
		StartTime:     time.Now(),
		customMetrics: make(map[string]*atomic.Int64),
	}
}

// RecordRequest records an HTTP request.
func (m *Metrics) RecordRequest(success bool) {
	m.TotalRequests.Add(1)
	if success {
		m.SuccessfulRequests.Add(1)
	} else {
		m.FailedRequests.Add(1)
	}
}

// RecordLogCreated records a created log entry.
func (m *Metrics) RecordLogCreated(count int) {
	m.LogsCreated.Add(int64(count))
}

// RecordLogQueried records a log query.
func (m *Metrics) RecordLogQueried(count int) {
	m.LogsQueried.Add(int64(count))
}

// RecordLogDeleted records a deleted log entry.
func (m *Metrics) RecordLogDeleted() {
	m.LogsDeleted.Add(1)
}

// RecordAlertCreated records a created alert.
func (m *Metrics) RecordAlertCreated() {
	m.AlertsCreated.Add(1)
}

// RecordAlertAcknowledged records an acknowledged alert.
func (m *Metrics) RecordAlertAcknowledged() {
	m.AlertsAcknowledged.Add(1)
}

// RecordAlertResolved records a resolved alert.
func (m *Metrics) RecordAlertResolved() {
	m.AlertsResolved.Add(1)
}

// RecordRuleCreated records a created rule.
func (m *Metrics) RecordRuleCreated() {
	m.RulesCreated.Add(1)
}

// RecordRuleUpdated records an updated rule.
func (m *Metrics) RecordRuleUpdated() {
	m.RulesUpdated.Add(1)
}

// RecordRuleDeleted records a deleted rule.
func (m *Metrics) RecordRuleDeleted() {
	m.RulesDeleted.Add(1)
}

// RecordSchedulerScan records a scheduler scan.
func (m *Metrics) RecordSchedulerScan(err error) {
	m.SchedulerScans.Add(1)
	if err != nil {
		m.SchedulerErrors.Add(1)
	}
}

// IncrCounter increments a custom counter.
func (m *Metrics) IncrCounter(name string, value int64) {
	m.mu.Lock()
	if m.customMetrics[name] == nil {
		m.customMetrics[name] = &atomic.Int64{}
	}
	m.mu.Unlock()
	m.customMetrics[name].Add(value)
}

// GetCounter returns the current value of a custom counter.
func (m *Metrics) GetCounter(name string) int64 {
	m.mu.RLock()
	counter := m.customMetrics[name]
	m.mu.RUnlock()
	if counter == nil {
		return 0
	}
	return counter.Load()
}

// Snapshot returns a snapshot of all metrics.
func (m *Metrics) Snapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := map[string]interface{}{
		"uptime":            time.Since(m.StartTime).String(),
		"total_requests":    m.TotalRequests.Load(),
		"successful_requests": m.SuccessfulRequests.Load(),
		"failed_requests":   m.FailedRequests.Load(),
		"logs_created":      m.LogsCreated.Load(),
		"logs_queried":      m.LogsQueried.Load(),
		"logs_deleted":      m.LogsDeleted.Load(),
		"alerts_created":    m.AlertsCreated.Load(),
		"alerts_acknowledged": m.AlertsAcknowledged.Load(),
		"alerts_resolved":   m.AlertsResolved.Load(),
		"rules_created":     m.RulesCreated.Load(),
		"rules_updated":     m.RulesUpdated.Load(),
		"rules_deleted":     m.RulesDeleted.Load(),
		"scheduler_scans":   m.SchedulerScans.Load(),
		"scheduler_errors":  m.SchedulerErrors.Load(),
	}

	for k, v := range m.customMetrics {
		snapshot["custom_"+k] = v.Load()
	}

	return snapshot
}

// Reset resets all metrics.
func (m *Metrics) Reset() {
	m.TotalRequests.Store(0)
	m.SuccessfulRequests.Store(0)
	m.FailedRequests.Store(0)
	m.LogsCreated.Store(0)
	m.LogsQueried.Store(0)
	m.LogsDeleted.Store(0)
	m.AlertsCreated.Store(0)
	m.AlertsAcknowledged.Store(0)
	m.AlertsResolved.Store(0)
	m.RulesCreated.Store(0)
	m.RulesUpdated.Store(0)
	m.RulesDeleted.Store(0)
	m.SchedulerScans.Store(0)
	m.SchedulerErrors.Store(0)
	m.StartTime = time.Now()

	m.mu.Lock()
	for _, v := range m.customMetrics {
		v.Store(0)
	}
	m.mu.Unlock()
}
