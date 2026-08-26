// Package logger provides structured logging for the logalert application.
// It offers a simple, efficient logging interface with level filtering,
// structured fields, and multiple output writers.
package logger

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// Logger is the main logger interface used throughout the application.
type Logger interface {
	// Debug logs a message at DEBUG level.
	Debug(msg string, args ...interface{})
	// Info logs a message at INFO level.
	Info(msg string, args ...interface{})
	// Warn logs a message at WARN level.
	Warn(msg string, args ...interface{})
	// Error logs a message at ERROR level.
	Error(msg string, args ...interface{})
	// Fatal logs a message at FATAL level and exits.
	Fatal(msg string, args ...interface{})
	// WithField returns a new Logger with the specified field added.
	WithField(key string, value interface{}) Logger
	// WithFields returns a new Logger with the specified fields added.
	WithFields(fields map[string]interface{}) Logger
	// WithContext returns a new Logger with context information.
	WithContext(ctx context.Context) Logger
	// SetLevel sets the minimum log level.
	SetLevel(level LogLevel)
	// GetLevel returns the current minimum log level.
	GetLevel() LogLevel
}

// defaultLogger is the global default logger instance.
var (
	defaultLogger Logger
	defaultOnce   sync.Once
)

// Default returns the global default logger instance.
func Default() Logger {
	defaultOnce.Do(func() {
		defaultLogger = NewLogger(LogLevelInfo, NewStdoutWriter())
	})
	return defaultLogger
}

// SetDefault sets the global default logger.
func SetDefault(l Logger) {
	defaultLogger = l
}

// Debug logs a message at DEBUG level using the default logger.
func Debug(msg string, args ...interface{}) {
	Default().Debug(msg, args...)
}

// Info logs a message at INFO level using the default logger.
func Info(msg string, args ...interface{}) {
	Default().Info(msg, args...)
}

// Warn logs a message at WARN level using the default logger.
func Warn(msg string, args ...interface{}) {
	Default().Warn(msg, args...)
}

// Error logs a message at ERROR level using the default logger.
func Error(msg string, args ...interface{}) {
	Default().Error(msg, args...)
}

// Fatal logs a message at FATAL level using the default logger.
func Fatal(msg string, args ...interface{}) {
	Default().Fatal(msg, args...)
}

// structuredLogger is the default implementation of Logger.
type structuredLogger struct {
	level  atomic.Int64
	writer LogWriter
	fields map[string]interface{}
	mu     sync.RWMutex
}

// NewLogger creates a new Logger with the specified level and writer.
func NewLogger(level LogLevel, writer LogWriter) Logger {
	l := &structuredLogger{
		writer: writer,
		fields: make(map[string]interface{}),
	}
	l.level.Store(int64(level))
	return l
}

// createEntry builds a LogEntry from the current logger state.
func (l *structuredLogger) createEntry(level LogLevel, msg string, args ...interface{}) *LogEntry {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   msg,
		Fields:    make(map[string]interface{}),
	}

	// Copy fields from logger
	l.mu.RLock()
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	l.mu.RUnlock()

	// Add additional fields from args (key-value pairs)
	for i := 0; i+1 < len(args); i += 2 {
		if key, ok := args[i].(string); ok {
			entry.Fields[key] = args[i+1]
		}
	}

	// Add caller information
	if _, file, line, ok := runtime.Caller(2); ok {
		entry.Caller = fmt.Sprintf("%s:%d", file, line)
	}

	return entry
}

// log performs the actual logging with level checking.
func (l *structuredLogger) log(level LogLevel, msg string, args ...interface{}) {
	if int64(level) < l.level.Load() {
		return
	}

	entry := l.createEntry(level, msg, args...)
	if err := l.writer.Write(entry); err != nil {
		// If writing fails, try stderr directly
		fmt.Fprintf(nil, "LOG ERROR: %v\n", err)
	}
}

// Debug logs a message at DEBUG level.
func (l *structuredLogger) Debug(msg string, args ...interface{}) {
	l.log(LogLevelDebug, msg, args...)
}

// Info logs a message at INFO level.
func (l *structuredLogger) Info(msg string, args ...interface{}) {
	l.log(LogLevelInfo, msg, args...)
}

// Warn logs a message at WARN level.
func (l *structuredLogger) Warn(msg string, args ...interface{}) {
	l.log(LogLevelWarn, msg, args...)
}

// Error logs a message at ERROR level.
func (l *structuredLogger) Error(msg string, args ...interface{}) {
	l.log(LogLevelError, msg, args...)
}

// Fatal logs a message at FATAL level.
func (l *structuredLogger) Fatal(msg string, args ...interface{}) {
	l.log(LogLevelFatal, msg, args...)
}

// WithField returns a new Logger with the specified field added.
func (l *structuredLogger) WithField(key string, value interface{}) Logger {
	newLogger := &structuredLogger{
		writer: l.writer,
		fields: make(map[string]interface{}),
	}
	newLogger.level.Store(l.level.Load())

	l.mu.RLock()
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	l.mu.RUnlock()

	newLogger.fields[key] = value
	return newLogger
}

// WithFields returns a new Logger with the specified fields added.
func (l *structuredLogger) WithFields(fields map[string]interface{}) Logger {
	newLogger := &structuredLogger{
		writer: l.writer,
		fields: make(map[string]interface{}),
	}
	newLogger.level.Store(l.level.Load())

	l.mu.RLock()
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	l.mu.RUnlock()

	for k, v := range fields {
		newLogger.fields[k] = v
	}
	return newLogger
}

// WithContext returns a new Logger with context information extracted.
func (l *structuredLogger) WithContext(ctx context.Context) Logger {
	newLogger := &structuredLogger{
		writer: l.writer,
		fields: make(map[string]interface{}),
	}
	newLogger.level.Store(l.level.Load())

	l.mu.RLock()
	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	l.mu.RUnlock()

	// Extract common context values
	if traceID, ok := ctx.Value("trace_id").(string); ok {
		newLogger.fields["trace_id"] = traceID
	}
	if requestID, ok := ctx.Value("request_id").(string); ok {
		newLogger.fields["request_id"] = requestID
	}
	if userID, ok := ctx.Value("user_id").(string); ok {
		newLogger.fields["user_id"] = userID
	}

	return newLogger
}

// SetLevel sets the minimum log level.
func (l *structuredLogger) SetLevel(level LogLevel) {
	l.level.Store(int64(level))
}

// GetLevel returns the current minimum log level.
func (l *structuredLogger) GetLevel() LogLevel {
	return LogLevel(l.level.Load())
}
