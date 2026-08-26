package timeutil

import (
	"time"
)

// TimeWindow represents a fixed-size time window for aggregation.
// It is defined by its start time and size.
type TimeWindow struct {
	// Start is the start time of the window.
	Start time.Time `json:"start"`
	// End is the end time of the window (exclusive).
	End time.Time `json:"end"`
	// Size is the duration of the window.
	Size time.Duration `json:"size"`
}

// NewTimeWindow creates a new TimeWindow with the given start time and size.
func NewTimeWindow(start time.Time, size time.Duration) *TimeWindow {
	normalizedStart := GlobalizeTime(start)
	return &TimeWindow{
		Start: normalizedStart,
		End:   normalizedStart.Add(size),
		Size:  size,
	}
}

// Contains checks if a time falls within the window [start, end).
func (w *TimeWindow) Contains(t time.Time) bool {
	normalizedT := GlobalizeTime(t)
	return !normalizedT.Before(w.Start) && normalizedT.Before(w.End)
}

// Overlaps checks if this window overlaps with another window.
func (w *TimeWindow) Overlaps(other *TimeWindow) bool {
	wStart := GlobalizeTime(w.Start)
	wEnd := GlobalizeTime(w.End)
	oStart := GlobalizeTime(other.Start)
	oEnd := GlobalizeTime(other.End)
	return wStart.Before(oEnd) && oStart.Before(wEnd)
}

// Duration returns the duration of the window.
func (w *TimeWindow) Duration() time.Duration {
	return w.Size
}

// TimeRange represents a user-defined time range for queries.
type TimeRange struct {
	// From is the start of the time range (inclusive).
	From time.Time `json:"from"`
	// To is the end of the time range (inclusive).
	To time.Time `json:"to"`
}

// NewTimeRange creates a new TimeRange.
func NewTimeRange(from, to time.Time) *TimeRange {
	normalizedFrom := GlobalizeTime(from)
	normalizedTo := GlobalizeTime(to)
	return &TimeRange{
		From: normalizedFrom,
		To:   normalizedTo,
	}
}

// Duration returns the duration of the time range.
func (r *TimeRange) Duration() time.Duration {
	return r.To.Sub(r.From)
}

// Contains checks if a time falls within the range [from, to].
func (r *TimeRange) Contains(t time.Time) bool {
	normalizedT := GlobalizeTime(t)
	return !normalizedT.Before(r.From) && !normalizedT.After(r.To)
}

// IsValid checks if the time range is valid (from <= to).
func (r *TimeRange) IsValid() bool {
	normalizedFrom := GlobalizeTime(r.From)
	normalizedTo := GlobalizeTime(r.To)
	return normalizedFrom.Before(normalizedTo) || normalizedFrom.Equal(normalizedTo)
}

// Overlaps checks if this range overlaps with another range.
func (r *TimeRange) Overlaps(other *TimeRange) bool {
	rFrom := GlobalizeTime(r.From)
	rTo := GlobalizeTime(r.To)
	oFrom := GlobalizeTime(other.From)
	oTo := GlobalizeTime(other.To)
	return rFrom.Before(oTo) && oFrom.Before(rTo)
}

// WindowIterator iterates over fixed-size time windows within a range.
type WindowIterator struct {
	range_     *TimeRange
	windowSize time.Duration
	current    time.Time
	done       bool
}

// NewWindowIterator creates a new WindowIterator.
func NewWindowIterator(range_ *TimeRange, windowSize time.Duration) *WindowIterator {
	normalizedFrom := GlobalizeTime(range_.From)
	normalizedTo := GlobalizeTime(range_.To)
	rangeCopy := &TimeRange{
		From: normalizedFrom,
		To:   normalizedTo,
	}
	return &WindowIterator{
		range_:     rangeCopy,
		windowSize: windowSize,
		current:    normalizedFrom,
		done:       false,
	}
}

// Next advances the iterator to the next window.
// Returns false when all windows have been exhausted.
func (it *WindowIterator) Next() bool {
	if it.done {
		return false
	}

	if it.current.After(it.range_.To) {
		it.done = true
		return false
	}

	return true
}

// Window returns the current window.
func (it *WindowIterator) Window() *TimeWindow {
	end := it.current.Add(it.windowSize)
	if end.After(it.range_.To) {
		end = it.range_.To.Add(time.Nanosecond)
	}

	window := NewTimeWindow(it.current, it.windowSize)
	window.End = end
	return window
}

// Advance moves the iterator to the next position.
func (it *WindowIterator) Advance() {
	it.current = it.current.Add(it.windowSize)
}

// DailyBuckets returns a list of daily time buckets for the given range.
func DailyBuckets(range_ *TimeRange) []*TimeWindow {
	normalizedFrom := GlobalizeTime(range_.From)
	normalizedTo := GlobalizeTime(range_.To)

	var buckets []*TimeWindow
	current := time.Date(normalizedFrom.Year(), normalizedFrom.Month(), normalizedFrom.Day(), 0, 0, 0, 0, time.UTC)

	for !current.After(normalizedTo) {
		window := NewTimeWindow(current, 24*time.Hour)
		buckets = append(buckets, window)
		current = current.Add(24 * time.Hour)
	}
	return buckets
}

// HourlyBuckets returns a list of hourly time buckets for the given range.
func HourlyBuckets(range_ *TimeRange) []*TimeWindow {
	normalizedFrom := GlobalizeTime(range_.From)
	normalizedTo := GlobalizeTime(range_.To)

	var buckets []*TimeWindow
	current := time.Date(normalizedFrom.Year(), normalizedFrom.Month(), normalizedFrom.Day(), normalizedFrom.Hour(), 0, 0, 0, time.UTC)

	for !current.After(normalizedTo) {
		window := NewTimeWindow(current, time.Hour)
		buckets = append(buckets, window)
		current = current.Add(time.Hour)
	}
	return buckets
}
