package logger

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// LogWriter is the interface for writing log entries to an output destination.
type LogWriter interface {
	// Write writes a log entry to the output.
	Write(entry *LogEntry) error
	// Close releases any resources held by the writer.
	Close() error
}

// StdoutWriter writes log entries to standard output.
type StdoutWriter struct {
	mu     sync.Mutex
	output io.Writer
}

// NewStdoutWriter creates a new StdoutWriter that writes to os.Stdout.
func NewStdoutWriter() *StdoutWriter {
	return &StdoutWriter{
		output: os.Stdout,
	}
}

// Write marshals the log entry to JSON and writes it to stdout.
func (w *StdoutWriter) Write(entry *LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	_, err = fmt.Fprintln(w.output, string(data))
	return err
}

// Close is a no-op for stdout writer.
func (w *StdoutWriter) Close() error {
	return nil
}

// FileWriter writes log entries to a file.
type FileWriter struct {
	mu     sync.Mutex
	file   *os.File
	closed bool
}

// NewFileWriter creates a new FileWriter that writes to the specified file path.
func NewFileWriter(path string) (*FileWriter, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file %s: %w", path, err)
	}
	return &FileWriter{
		file: f,
	}, nil
}

// Write marshals the log entry to JSON and writes it to the file.
func (w *FileWriter) Write(entry *LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return fmt.Errorf("file writer is closed")
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("failed to marshal log entry: %w", err)
	}

	_, err = fmt.Fprintln(w.file, string(data))
	return err
}

// Close closes the file writer.
func (w *FileWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.closed {
		return nil
	}
	w.closed = true
	return w.file.Close()
}

// DiscardWriter discards all log entries. Useful for testing.
type DiscardWriter struct{}

// NewDiscardWriter creates a new DiscardWriter.
func NewDiscardWriter() *DiscardWriter {
	return &DiscardWriter{}
}

// Write discards the log entry.
func (w *DiscardWriter) Write(entry *LogEntry) error {
	return nil
}

// Close is a no-op.
func (w *DiscardWriter) Close() error {
	return nil
}

// BufferWriter writes log entries to an in-memory buffer for later retrieval.
type BufferWriter struct {
	mu     sync.RWMutex
	buffer []*LogEntry
	maxSize int
}

// NewBufferWriter creates a new BufferWriter with the specified maximum buffer size.
func NewBufferWriter(maxSize int) *BufferWriter {
	if maxSize <= 0 {
		maxSize = 10000
	}
	return &BufferWriter{
		buffer:  make([]*LogEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

// Write appends the log entry to the buffer.
// If the buffer is full, the oldest entry is removed.
func (w *BufferWriter) Write(entry *LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if len(w.buffer) >= w.maxSize {
		// Remove the oldest entry to make room
		w.buffer = w.buffer[1:]
	}
	w.buffer = append(w.buffer, entry)
	return nil
}

// Close is a no-op for buffer writer.
func (w *BufferWriter) Close() error {
	return nil
}

// Entries returns a copy of all buffered log entries.
func (w *BufferWriter) Entries() []*LogEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	result := make([]*LogEntry, len(w.buffer))
	copy(result, w.buffer)
	return result
}

// EntriesSince returns all entries after the specified time.
func (w *BufferWriter) EntriesSince(t time.Time) []*LogEntry {
	w.mu.RLock()
	defer w.mu.RUnlock()

	var result []*LogEntry
	for _, entry := range w.buffer {
		if entry.Timestamp.After(t) {
			result = append(result, entry)
		}
	}
	return result
}

// Clear removes all buffered entries.
func (w *BufferWriter) Clear() {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.buffer = w.buffer[:0]
}

// MultiWriter writes to multiple LogWriters simultaneously.
type MultiWriter struct {
	writers []LogWriter
}

// NewMultiWriter creates a new MultiWriter that distributes writes to all provided writers.
func NewMultiWriter(writers ...LogWriter) *MultiWriter {
	return &MultiWriter{
		writers: writers,
	}
}

// Write writes the log entry to all underlying writers.
// Returns the first error encountered, if any.
func (w *MultiWriter) Write(entry *LogEntry) error {
	for _, writer := range w.writers {
		if err := writer.Write(entry); err != nil {
			return err
		}
	}
	return nil
}

// Close closes all underlying writers.
func (w *MultiWriter) Close() error {
	var firstErr error
	for _, writer := range w.writers {
		if err := writer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}
