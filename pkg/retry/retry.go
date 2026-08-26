// Package retry provides retry logic for operations that may temporarily fail.
package retry

import (
	"context"
	"fmt"
	"math"
	"time"
)

// Config holds retry configuration.
type Config struct {
	// MaxAttempts is the maximum number of attempts (including the initial one).
	MaxAttempts int
	// InitialDelay is the delay before the first retry.
	InitialDelay time.Duration
	// MaxDelay is the maximum delay between retries.
	MaxDelay time.Duration
	// BackoffMultiplier is the multiplier for exponential backoff.
	BackoffMultiplier float64
	// RetryableFunc determines if an error is retryable.
	RetryableFunc func(error) bool
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		MaxAttempts:       3,
		InitialDelay:      100 * time.Millisecond,
		MaxDelay:          5 * time.Second,
		BackoffMultiplier: 2.0,
		RetryableFunc:     func(err error) bool { return err != nil },
	}
}

// Do executes the operation with retries.
func Do(ctx context.Context, cfg *Config, operation func() error) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var lastErr error
	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			backoff := float64(cfg.InitialDelay) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
			if backoff > float64(cfg.MaxDelay) {
				backoff = float64(cfg.MaxDelay)
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(backoff)):
			}
		}

		lastErr = operation()
		if lastErr == nil {
			return nil
		}

		if !cfg.RetryableFunc(lastErr) {
			return lastErr
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// DoWithValue executes an operation that returns a value, with retries.
func DoWithValue[T any](ctx context.Context, cfg *Config, operation func() (T, error)) (T, error) {
	var result T
	err := Do(ctx, cfg, func() error {
		var opErr error
		result, opErr = operation()
		return opErr
	})
	return result, err
}
