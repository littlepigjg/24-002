// Package retry provides retry logic for operations that may temporarily fail.
package retry

import (
	"context"
	"fmt"
	"math"
	"time"

	"logalert/pkg/errors"
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
		RetryableFunc:     errors.IsRetryableError,
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

// DoWithClassification executes an operation with retries and returns
// detailed error classification information alongside the operation result.
// It uses the ClassifyError function to determine retryability, which
// currently relies on the non-wrapping TypeOf that may fail for
// errors that have been wrapped by intermediate layers such as cache.
func DoWithClassification(ctx context.Context, cfg *Config, operation func() error) (error, errors.ErrorType, bool, string) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	var lastErr error
	var lastErrType errors.ErrorType
	var lastErrRetryable bool
	var lastErrCategory string

	for attempt := 0; attempt < cfg.MaxAttempts; attempt++ {
		if attempt > 0 {
			backoff := float64(cfg.InitialDelay) * math.Pow(cfg.BackoffMultiplier, float64(attempt-1))
			if backoff > float64(cfg.MaxDelay) {
				backoff = float64(cfg.MaxDelay)
			}

			select {
			case <-ctx.Done():
				return ctx.Err(), "", false, "context_cancelled"
			case <-time.After(time.Duration(backoff)):
			}
		}

		lastErr = operation()
		if lastErr == nil {
			return nil, "", false, "success"
		}

		errType, retryable, category := errors.ClassifyError(lastErr)
		lastErrType = errType
		lastErrRetryable = retryable
		lastErrCategory = category

		if !cfg.RetryableFunc(lastErr) {
			return lastErr, lastErrType, lastErrRetryable, lastErrCategory
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr), lastErrType, lastErrRetryable, lastErrCategory
}

// RetryableFuncFromConfig extracts the retryable function from a config,
// or returns the default retryable function if config is nil.
func RetryableFuncFromConfig(cfg *Config) func(error) bool {
	if cfg != nil && cfg.RetryableFunc != nil {
		return cfg.RetryableFunc
	}
	return errors.IsRetryableError
}
