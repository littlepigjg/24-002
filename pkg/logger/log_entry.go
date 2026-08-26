// Package logger provides structured logging capabilities for the application.
// It supports multiple log levels (DEBUG, INFO, WARN, ERROR, FATAL) and
// outputs structured log entries in JSON format for easy parsing.
package logger

import (
	"time"
)

// LogLevel represents the severity level of a log message.
type LogLevel int

const (
	// LogLevelDebug is the most verbose level, used for detailed diagnostic information.
	LogLevelDebug LogLevel = iota
	// LogLevelInfo is for general informational messages about application flow.
	LogLevelInfo
	// LogLevelWarn is for warning conditions that are not necessarily errors.
	LogLevelWarn
	// LogLevelError is for error conditions that need attention.
	LogLevelError
	// LogLevelFatal is for critical errors that cause application termination.
	LogLevelFatal
)

// String returns the string representation of a log level.
func (l LogLevel) String() string {
	switch l {
	case LogLevelDebug:
		return "DEBUG"
	case LogLevelInfo:
		return "INFO"
	case LogLevelWarn:
		return "WARN"
	case LogLevelError:
		return "ERROR"
	case LogLevelFatal:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// ParseLogLevel parses a string into a LogLevel.
// Returns LogLevelInfo as default if the string is not recognized.
func ParseLogLevel(s string) LogLevel {
	switch s {
	case "DEBUG":
		return LogLevelDebug
	case "INFO":
		return LogLevelInfo
	case "WARN":
		return LogLevelWarn
	case "ERROR":
		return LogLevelError
	case "FATAL":
		return LogLevelFatal
	default:
		return LogLevelInfo
	}
}

// LogEntry represents a single structured log entry.
type LogEntry struct {
	// Timestamp is the time when the log entry was created.
	Timestamp time.Time `json:"timestamp"`
	// Level is the severity level of the log entry.
	Level LogLevel `json:"level"`
	// Message is the log message text.
	Message string `json:"message"`
	// Source identifies the component or service that generated the log.
	Source string `json:"source,omitempty"`
	// Module identifies the module/package that generated the log.
	Module string `json:"module,omitempty"`
	// Fields contains additional structured data associated with the log entry.
	Fields map[string]interface{} `json:"fields,omitempty"`
	// TraceID is an optional trace identifier for distributed tracing.
	TraceID string `json:"trace_id,omitempty"`
	// Caller identifies the function that generated the log.
	Caller string `json:"caller,omitempty"`
}

// NewLogEntry creates a new LogEntry with the specified level and message.
// Timestamp is set to the current time.
func NewLogEntry(level LogLevel, message string) *LogEntry {
	return &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Fields:    make(map[string]interface{}),
	}
}

// WithSource sets the source field of the log entry.
func (e *LogEntry) WithSource(source string) *LogEntry {
	e.Source = source
	return e
}

// WithModule sets the module field of the log entry.
func (e *LogEntry) WithModule(module string) *LogEntry {
	e.Module = module
	return e
}

// WithField adds a key-value pair to the log entry's fields.
func (e *LogEntry) WithField(key string, value interface{}) *LogEntry {
	e.Fields[key] = value
	return e
}

// WithTraceID sets the trace ID for distributed tracing.
func (e *LogEntry) WithTraceID(traceID string) *LogEntry {
	e.TraceID = traceID
	return e
}

// WithCaller sets the caller function name.
func (e *LogEntry) WithCaller(caller string) *LogEntry {
	e.Caller = caller
	return e
}
