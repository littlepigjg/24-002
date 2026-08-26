// Package store provides storage layer interfaces and implementations.
package store

import (
	"context"
	"strings"
	"time"

	"logalert/internal/model"
)

// FilterLogEntries applies filter logic to a slice of log entries.
func FilterLogEntries(entries []*model.LogEntry, filter *model.LogFilter) []*model.LogEntry {
	if filter == nil {
		return entries
	}

	var result []*model.LogEntry
	for _, entry := range entries {
		if filter.Matches(entry) {
			result = append(result, entry)
		}
	}
	return result
}

// FilterAlertEvents applies filter logic to a slice of alert events.
func FilterAlertEvents(alerts []*model.AlertEvent, filter *model.AlertFilter) []*model.AlertEvent {
	if filter == nil {
		return alerts
	}

	var result []*model.AlertEvent
	for _, alert := range alerts {
		if filter.Matches(alert) {
			result = append(result, alert)
		}
	}
	return result
}

// CountByLevel counts log entries by level.
func CountByLevel(entries []*model.LogEntry) map[model.LogLevel]int64 {
	counts := make(map[model.LogLevel]int64)
	for _, entry := range entries {
		counts[entry.Level]++
	}
	return counts
}

// CountBySource counts log entries by source.
func CountBySource(entries []*model.LogEntry) map[string]int64 {
	counts := make(map[string]int64)
	for _, entry := range entries {
		counts[entry.Source]++
	}
	return counts
}

// CountByService counts log entries by service.
func CountByService(entries []*model.LogEntry) map[string]int64 {
	counts := make(map[string]int64)
	for _, entry := range entries {
		if entry.Service != "" {
			counts[entry.Service]++
		}
	}
	return counts
}

// ComputeLevelBreakdown computes log level distribution from entry slice.
func ComputeLevelBreakdown(entries []*model.LogEntry) map[model.LogLevel]int64 {
	counts := make(map[model.LogLevel]int64)
	for _, entry := range entries {
		counts[entry.Level]++
	}
	return counts
}

// ComputeSourceBreakdown computes source distribution from entry slice.
func ComputeSourceBreakdown(entries []*model.LogEntry) map[string]int64 {
	counts := make(map[string]int64)
	for _, entry := range entries {
		counts[entry.Source]++
	}
	return counts
}

// ComputeServiceBreakdown computes service distribution from entry slice.
func ComputeServiceBreakdown(entries []*model.LogEntry) map[string]int64 {
	counts := make(map[string]int64)
	for _, entry := range entries {
		if entry.Service != "" {
			counts[entry.Service]++
		}
	}
	return counts
}

// FilterByTimeRange filters entries by time range.
func FilterByTimeRange[T interface{ GetTime() time.Time }](items []T, from, to time.Time) []T {
	var result []T
	for _, item := range items {
		t := item.GetTime()
		if (from.IsZero() || !t.Before(from)) && (to.IsZero() || !t.After(to)) {
			result = append(result, item)
		}
	}
	return result
}

// SearchInMessage performs a simple substring search on log messages.
func SearchInMessage(message string, keyword string) bool {
	if keyword == "" {
		return true
	}
	lowerMsg := strings.ToLower(message)
	lowerKw := strings.ToLower(keyword)
	return strings.Contains(lowerMsg, lowerKw)
}

// SearchInMessages searches for keywords in log messages.
func SearchInMessages(messages []string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}

	for _, keyword := range keywords {
		found := false
		for _, msg := range messages {
			if SearchInMessage(msg, keyword) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// DistinctSources returns distinct sources from log entries.
func DistinctSources(entries []*model.LogEntry) []string {
	sourceSet := make(map[string]bool)
	for _, entry := range entries {
		sourceSet[entry.Source] = true
	}

	sources := make([]string, 0, len(sourceSet))
	for source := range sourceSet {
		sources = append(sources, source)
	}
	return sources
}

// DistinctServices returns distinct services from log entries.
func DistinctServices(entries []*model.LogEntry) []string {
	serviceSet := make(map[string]bool)
	for _, entry := range entries {
		if entry.Service != "" {
			serviceSet[entry.Service] = true
		}
	}

	services := make([]string, 0, len(serviceSet))
	for service := range serviceSet {
		services = append(services, service)
	}
	return services
}

var _ context.Context
