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
