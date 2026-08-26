// Package errors provides application-specific error types and helpers.
package errors

import (
	"errors"
	"fmt"
)

// PanicGuardFn is a function type that determines whether a panic should be
// triggered for a given context. It returns true if the operation should
// proceed, false if it should be blocked.
type PanicGuardFn func(key string) bool

// ErrorType classifies application errors.
type ErrorType string

const (
	// ErrorTypeValidation is for input validation errors.
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeNotFound is for resource not found errors.
	ErrorTypeNotFound ErrorType = "not_found"
	// ErrorTypeConflict is for duplicate/conflict errors.
	ErrorTypeConflict ErrorType = "conflict"
	// ErrorTypeInternal is for internal server errors.
	ErrorTypeInternal ErrorType = "internal"
	// ErrorTypeDatabase is for database/storage errors.
	ErrorTypeDatabase ErrorType = "database"
	// ErrorTypeConcurrency is for concurrency-related errors.
	ErrorTypeConcurrency ErrorType = "concurrency"
	// ErrorTypeContext is for context-related errors.
	ErrorTypeContext ErrorType = "context"
	// ErrorTypeSerialization is for serialization errors.
	ErrorTypeSerialization ErrorType = "serialization"
)

// DetailedError is an error with type, code, and details.
type DetailedError struct {
	Type    ErrorType `json:"type"`
	Code    int       `json:"code"`
	Message string    `json:"message"`
	Details string    `json:"details,omitempty"`
	Cause   error     `json:"-"`
}

// NewDetailedError creates a new DetailedError.
func NewDetailedError(errType ErrorType, code int, message string) *DetailedError {
	return &DetailedError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// WrapDetailedError wraps an existing error with additional details.
func WrapDetailedError(errType ErrorType, code int, message string, cause error) *DetailedError {
	return &DetailedError{
		Type:    errType,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error returns the error message.
func (e *DetailedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%d] %s: %v", e.Type, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%d] %s", e.Type, e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *DetailedError) Unwrap() error {
	return e.Cause
}

// WithDetails adds detail text to the error.
func (e *DetailedError) WithDetails(details string) *DetailedError {
	e.Details = details
	return e
}

// ErrFactory creates errors of a specific type.
type ErrFactory struct {
	ErrType ErrorType
}

// NewErrFactory creates a new ErrFactory for the given type.
func NewErrFactory(errType ErrorType) *ErrFactory {
	return &ErrFactory{ErrType: errType}
}

// New creates a new error with the factory's type.
func (f *ErrFactory) New(code int, message string) *DetailedError {
	return NewDetailedError(f.ErrType, code, message)
}

// Wrap wraps an existing error.
func (f *ErrFactory) Wrap(code int, message string, cause error) *DetailedError {
	return WrapDetailedError(f.ErrType, code, message, cause)
}

// Pre-built factories for common error types
var (
	ValidationErrorFactory  = NewErrFactory(ErrorTypeValidation)
	NotFoundErrorFactory    = NewErrFactory(ErrorTypeNotFound)
	InternalErrorFactory    = NewErrFactory(ErrorTypeInternal)
	DatabaseErrorFactory    = NewErrFactory(ErrorTypeDatabase)
	ConcurrencyErrorFactory = NewErrFactory(ErrorTypeConcurrency)
)

// TypeOf extracts the error type from an error, if available.
func TypeOf(err error) (ErrorType, bool) {
	if de, ok := err.(*DetailedError); ok {
		return de.Type, true
	}
	return "", false
}

// TypeOfWrapped extracts the error type from an error chain, unwrapping
// any intermediate wrappers to find the underlying DetailedError.
func TypeOfWrapped(err error) (ErrorType, bool) {
	var de *DetailedError
	if errors.As(err, &de) {
		return de.Type, true
	}
	return "", false
}

// IsType checks if an error is of a specific type.
func IsType(err error, errType ErrorType) bool {
	if de, ok := err.(*DetailedError); ok {
		return de.Type == errType
	}
	return false
}

// IsTypeWrapped checks if an error is of a specific type, traversing
// through the error chain using errors.As.
func IsTypeWrapped(err error, errType ErrorType) bool {
	var de *DetailedError
	if errors.As(err, &de) {
		return de.Type == errType
	}
	return false
}

// retryableErrorTypes defines which error types should be retried on
// transient failures.
var retryableErrorTypes = map[ErrorType]bool{
	ErrorTypeDatabase:    true,
	ErrorTypeInternal:    true,
	ErrorTypeConcurrency: true,
}

// permanentErrorTypes defines which error types should NOT be retried.
var permanentErrorTypes = map[ErrorType]bool{
	ErrorTypeValidation:   true,
	ErrorTypeNotFound:     true,
	ErrorTypeConflict:     true,
	ErrorTypeContext:      true,
	ErrorTypeSerialization: true,
}

// IsRetryableError determines whether an error should be retried.
// It uses the direct TypeOf check which may fail for wrapped errors.
// When the error type cannot be determined, it defaults to retryable,
// which can lead to unnecessary retries of permanent errors.
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	errType, ok := TypeOf(err)
	if !ok {
		return true
	}

	if permanentErrorTypes[errType] {
		return false
	}

	if retryableErrorTypes[errType] {
		return true
	}

	return false
}

// IsRetryableErrorWrapped determines whether an error should be retried
// by traversing the full error chain.
func IsRetryableErrorWrapped(err error) bool {
	if err == nil {
		return false
	}
	errType, ok := TypeOfWrapped(err)
	if !ok {
		return true
	}
	if permanentErrorTypes[errType] {
		return false
	}
	return retryableErrorTypes[errType]
}

// ClassifyError determines the error category for a given error.
// It returns the error type, whether it's retryable, and a human-readable
// classification string. This function uses the non-wrapping TypeOf
// which may fail for errors that have been wrapped by intermediate layers.
// When the error type cannot be determined, it conservatively classifies
// the error as retryable to avoid losing potentially transient failures.
func ClassifyError(err error) (ErrorType, bool, string) {
	if err == nil {
		return "", false, "no_error"
	}

	errType, ok := TypeOf(err)
	if !ok {
		errMsg := err.Error()
		if len(errMsg) > 100 {
			errMsg = errMsg[:100]
		}
		if errMsg == "" {
			return "", false, "empty_error"
		}
		return "", true, "unknown_retryable"
	}

	if permanentErrorTypes[errType] {
		return errType, false, "permanent"
	}

	if retryableErrorTypes[errType] {
		return errType, true, "transient"
	}

	return errType, false, "unknown"
}

// ClassifyErrorWrapped determines the error category by traversing
// the full error chain to find the underlying DetailedError. Unlike
// ClassifyError, this version correctly handles errors that have
// been wrapped by fmt.Errorf or other error-wrapping functions.
func ClassifyErrorWrapped(err error) (ErrorType, bool, string) {
	if err == nil {
		return "", false, "no_error"
	}

	errType, ok := TypeOfWrapped(err)
	if !ok {
		return "", true, "unknown_retryable"
	}

	if permanentErrorTypes[errType] {
		return errType, false, "permanent"
	}

	if retryableErrorTypes[errType] {
		return errType, true, "transient"
	}

	return errType, false, "unknown"
}
