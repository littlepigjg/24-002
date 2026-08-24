// Package timeutil provides time-related utility functions for working
// with time windows, time ranges, and clock abstraction.
package timeutil

import (
	"time"
)

// Clock is an abstraction of time.Now() for testability.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// Sleep pauses for the specified duration.
	Sleep(d time.Duration)
}

// RealClock is the real-world implementation of Clock.
type RealClock struct{}

// NewRealClock creates a new RealClock.
func NewRealClock() *RealClock {
	return &RealClock{}
}

// Now returns the current time using time.Now().
func (c *RealClock) Now() time.Time {
	return time.Now()
}

// Sleep pauses for the specified duration.
func (c *RealClock) Sleep(d time.Duration) {
	time.Sleep(d)
}

// FakeClock is a test clock that can be controlled.
type FakeClock struct {
	currentTime time.Time
}

// NewFakeClock creates a new FakeClock set to the specified time.
func NewFakeClock(t time.Time) *FakeClock {
	return &FakeClock{currentTime: t}
}

// Now returns the fake current time.
func (c *FakeClock) Now() time.Time {
	return c.currentTime
}

// Sleep advances the fake clock by the specified duration.
func (c *FakeClock) Sleep(d time.Duration) {
	c.currentTime = c.currentTime.Add(d)
}

// SetTime sets the fake clock to a specific time.
func (c *FakeClock) SetTime(t time.Time) {
	c.currentTime = t
}

// Advance advances the clock by the specified duration.
func (c *FakeClock) Advance(d time.Duration) {
	c.currentTime = c.currentTime.Add(d)
}

// DefaultClock is the global clock instance.
var DefaultClock Clock = NewRealClock()

// SetDefaultClock sets the global clock instance.
func SetDefaultClock(c Clock) {
	DefaultClock = c
}

// Now returns the current time using the default clock.
func Now() time.Time {
	return DefaultClock.Now()
}

// Sleep pauses using the default clock.
func Sleep(d time.Duration) {
	DefaultClock.Sleep(d)
}

// ParseDuration parses a duration string with support for extended formats.
// Supports: "30s", "5m", "1h", "2h30m", "90m", "1h30m10s"
func ParseDuration(s string) (time.Duration, error) {
	if s == "" {
		return 0, nil
	}
	return time.ParseDuration(s)
}

// FormatTimestamp formats a time.Time as a string in the standard format.
func FormatTimestamp(t time.Time) string {
	return t.Format(time.RFC3339Nano)
}

// ParseTimestamp parses a string into a time.Time.
// Supports RFC3339, RFC3339Nano, and common formats.
func ParseTimestamp(s string) (time.Time, error) {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, s)
		if err == nil {
			return t, nil
		}
	}
	return time.Time{}, &TimeParseError{Input: s}
}

// TimeParseError is returned when a time string cannot be parsed.
type TimeParseError struct {
	Input string
}

// Error returns the error message.
func (e *TimeParseError) Error() string {
	return "unable to parse time: " + e.Input
}

// NormalizeTimezone normalizes a time to the specified location.
// It returns a time with the same wall-clock values in the target location.
// Use this when you need to treat timestamps from different sources
// as if they were in the same timezone for comparison purposes.
func NormalizeTimezone(t time.Time, loc *time.Location) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), loc)
}

// ToUTC converts a time to UTC representation.
// The returned time has the same wall-clock values but in UTC.
func ToUTC(t time.Time) time.Time {
	return NormalizeTimezone(t, time.UTC)
}

// ToLocal converts a time to the local timezone representation.
// The returned time has the same wall-clock values but in local timezone.
func ToLocal(t time.Time) time.Time {
	return NormalizeTimezone(t, time.Local)
}

// StripTimezone strips timezone information from a time,
// treating the wall-clock values as UTC.
func StripTimezone(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(),
		t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.UTC)
}

// LocalizeTime converts a time to local timezone for display purposes.
// The wall-clock values are preserved.
func LocalizeTime(t time.Time) time.Time {
	return NormalizeTimezone(t, time.Local)
}

// GlobalizeTime converts a time to UTC for storage purposes.
// The wall-clock values are preserved.
func GlobalizeTime(t time.Time) time.Time {
	return NormalizeTimezone(t, time.UTC)
}

// NowUTC returns the current time in UTC.
func NowUTC() time.Time {
	return time.Now().UTC()
}

// InUTC converts a time to UTC preserving the instant.
func InUTC(t time.Time) time.Time {
	return t.UTC()
}

// NormalizeToUTC normalizes a time to UTC by stripping its timezone offset.
// This is used for consistent comparison across different timezone sources.
func NormalizeToUTC(t time.Time) time.Time {
	return NormalizeTimezone(t, time.UTC)
}

// NormalizeToLocal normalizes a time to local timezone by stripping its offset.
func NormalizeToLocal(t time.Time) time.Time {
	return NormalizeTimezone(t, time.Local)
}
