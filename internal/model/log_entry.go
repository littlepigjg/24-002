// Package model defines the core data structures for the logalert application.
package model

import (
	"time"
)

// LogLevel represents the severity level of a log entry.
type LogLevel string

const (
	// LevelInfo is for informational messages.
	LevelInfo LogLevel = "INFO"
	// LevelWarn is for warning messages.
	LevelWarn LogLevel = "WARN"
	// LevelError is for error messages.
	LevelError LogLevel = "ERROR"
	// LevelDebug is for debug messages.
	LevelDebug LogLevel = "DEBUG"
	// LevelFatal is for fatal messages.
	LevelFatal LogLevel = "FATAL"
)

// AllLogLevels returns all valid log levels.
func AllLogLevels() []LogLevel {
	return []LogLevel{LevelDebug, LevelInfo, LevelWarn, LevelError, LevelFatal}
}

// LogEntry represents a log entry received from a service.
type LogEntry struct {
	// ID is the unique identifier for this log entry.
	ID string `json:"id"`
	// Timestamp is when the log entry was created.
	Timestamp time.Time `json:"timestamp"`
	// Level is the severity level of the log.
	Level LogLevel `json:"level"`
	// Source is the service/component that generated the log.
	Source string `json:"source"`
	// Message is the log message.
	Message string `json:"message"`
	// Service is the service name.
	Service string `json:"service,omitempty"`
	// Tags are additional metadata tags.
	Tags map[string]string `json:"tags,omitempty"`
	// Keywords are extracted keywords for searching.
	Keywords []string `json:"keywords,omitempty"`
	// ReceivedAt is when the log was received by the system.
	ReceivedAt time.Time `json:"received_at"`
}

// NewLogEntry creates a new LogEntry with sensible defaults.
func NewLogEntry(source string, level LogLevel, message string) *LogEntry {
	return &LogEntry{
		ID:        GenerateID(),
		Timestamp: time.Now(),
		Level:     level,
		Source:    source,
		Message:   message,
		Tags:      make(map[string]string),
		ReceivedAt: time.Now(),
	}
}

// WithTimestamp sets the timestamp of the log entry.
func (e *LogEntry) WithTimestamp(t time.Time) *LogEntry {
	e.Timestamp = t
	return e
}

// WithService sets the service name.
func (e *LogEntry) WithService(service string) *LogEntry {
	e.Service = service
	return e
}

// WithTags sets the tags for the log entry.
func (e *LogEntry) WithTags(tags map[string]string) *LogEntry {
	e.Tags = tags
	return e
}

// HasTag checks if a tag exists.
func (e *LogEntry) HasTag(key string) bool {
	if e.Tags == nil {
		return false
	}
	_, ok := e.Tags[key]
	return ok
}

// GetTag returns the value of a tag.
func (e *LogEntry) GetTag(key string) (string, bool) {
	if e.Tags == nil {
		return "", false
	}
	val, ok := e.Tags[key]
	return val, ok
}

// IsLevel checks if the log entry has a specific level.
func (e *LogEntry) IsLevel(level LogLevel) bool {
	return e.Level == level
}

// IsError checks if this is an error-level log entry.
func (e *LogEntry) IsError() bool {
	return e.Level == LevelError || e.Level == LevelFatal
}

// ToMap converts the log entry to a map for logging.
func (e *LogEntry) ToMap() map[string]interface{} {
	return map[string]interface{}{
		"id":        e.ID,
		"timestamp": e.Timestamp,
		"level":     e.Level,
		"source":    e.Source,
		"message":   e.Message,
		"service":   e.Service,
		"tags":      e.Tags,
	}
}

// LogFilter defines criteria for filtering log entries.
type LogFilter struct {
	// Levels to include (empty means all levels).
	Levels []LogLevel `json:"levels,omitempty"`
	// Sources to include (empty means all sources).
	Sources []string `json:"sources,omitempty"`
	// Keywords to search for in messages.
	Keywords []string `json:"keywords,omitempty"`
	// StartTime is the earliest timestamp to include.
	StartTime *time.Time `json:"start_time,omitempty"`
	// EndTime is the latest timestamp to include.
	EndTime *time.Time `json:"end_time,omitempty"`
	// Service filter.
	Service string `json:"service,omitempty"`
	// Tags filter.
	Tags map[string]string `json:"tags,omitempty"`
}

// Matches checks if a log entry matches the filter.
func (f *LogFilter) Matches(entry *LogEntry) bool {
	// Check level
	if len(f.Levels) > 0 {
		levelMatch := false
		for _, l := range f.Levels {
			if l == entry.Level {
				levelMatch = true
				break
			}
		}
		if !levelMatch {
			return false
		}
	}

	// Check source
	if len(f.Sources) > 0 {
		sourceMatch := false
		for _, s := range f.Sources {
			if s == entry.Source {
				sourceMatch = true
				break
			}
		}
		if !sourceMatch {
			return false
		}
	}

	// Check service
	if f.Service != "" && f.Service != entry.Service {
		return false
	}

	// Check time range
	if f.StartTime != nil && entry.Timestamp.Before(*f.StartTime) {
		return false
	}
	if f.EndTime != nil && entry.Timestamp.After(*f.EndTime) {
		return false
	}

	// Check keywords
	if len(f.Keywords) > 0 {
		for _, kw := range f.Keywords {
			if !containsKeyword(entry.Message, kw) {
				return false
			}
		}
	}

	// Check tags
	for k, v := range f.Tags {
		ev, ok := entry.Tags[k]
		if !ok || ev != v {
			return false
		}
	}

	return true
}

// containsKeyword checks if a message contains a keyword (case-insensitive).
func containsKeyword(message, keyword string) bool {
	if keyword == "" {
		return true
	}
	lower := toLower(message)
	lowerKw := toLower(keyword)
	return len(lower) >= len(lowerKw) && searchString(lower, lowerKw)
}

// toLower converts a string to lowercase (simple version for ASCII).
func toLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

// searchString searches for substr in s (simple implementation).
func searchString(s, substr string) bool {
	if len(substr) == 0 {
		return true
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
