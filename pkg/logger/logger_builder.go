// Package logger provides structured logging for the logalert application.
package logger

import (
	"io"
	"os"
	"sync"
)

// LoggerBuilder provides a builder pattern for constructing Logger instances.
type LoggerBuilder struct {
	level  LogLevel
	writer LogWriter
	fields map[string]interface{}
}

// NewLoggerBuilder creates a new LoggerBuilder with default settings.
func NewLoggerBuilder() *LoggerBuilder {
	return &LoggerBuilder{
		level:  LogLevelInfo,
		writer: NewStdoutWriter(),
		fields: make(map[string]interface{}),
	}
}

// SetLevel sets the minimum log level.
func (b *LoggerBuilder) SetLevel(level LogLevel) *LoggerBuilder {
	b.level = level
	return b
}

// SetWriter sets the log output writer.
func (b *LoggerBuilder) SetWriter(w LogWriter) *LoggerBuilder {
	b.writer = w
	return b
}

// SetOutputFile sets the output to a file.
func (b *LoggerBuilder) SetOutputFile(path string) *LoggerBuilder {
	fw, err := NewFileWriter(path)
	if err != nil {
		return b
	}
	b.writer = fw
	return b
}

// AddField adds a field that will be included in all log entries.
func (b *LoggerBuilder) AddField(key string, value interface{}) *LoggerBuilder {
	b.fields[key] = value
	return b
}

// Build creates the Logger instance.
func (b *LoggerBuilder) Build() Logger {
	logger := NewLogger(b.level, b.writer)

	for k, v := range b.fields {
		logger = logger.WithField(k, v)
	}

	return logger
}

// ConsoleLogger creates a logger with console output.
func ConsoleLogger(level LogLevel) Logger {
	return NewLogger(level, NewStdoutWriter())
}

// FileLogger creates a logger that writes to a file.
func FileLogger(level LogLevel, path string) (Logger, error) {
	fw, err := NewFileWriter(path)
	if err != nil {
		return nil, err
	}
	return NewLogger(level, fw), nil
}

// MultiOutputLogger creates a logger that writes to multiple outputs.
func MultiOutputLogger(level LogLevel, paths ...string) (Logger, error) {
	var writers []LogWriter
	writers = append(writers, NewStdoutWriter())

	for _, path := range paths {
		fw, err := NewFileWriter(path)
		if err != nil {
			return nil, err
		}
		writers = append(writers, fw)
	}

	return NewLogger(level, NewMultiWriter(writers...)), nil
}

// LogEntryFormatter formats log entries for output.
type LogEntryFormatter struct {
	mu       sync.Mutex
	template string
}

// NewLogEntryFormatter creates a new formatter with a template.
func NewLogEntryFormatter(template string) *LogEntryFormatter {
	return &LogEntryFormatter{
		template: template,
	}
}

// Format formats a log entry as a string.
func (f *LogEntryFormatter) Format(entry *LogEntry) string {
	f.mu.Lock()
	defer f.mu.Unlock()

	// Simple format: timestamp [LEVEL] source - message
	return entry.Timestamp.Format("2006-01-02 15:04:05.000") + " [" + entry.Level.String() + "] " +
		entry.Source + " - " + entry.Message
}

// Ensure unused imports don't cause issues
var _ io.Writer
var _ = os.Stdout
