// Package errors provides application-specific error types and helpers.
package errors

import (
	"fmt"
)

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
	Type     ErrorType `json:"type"`
	Code     int       `json:"code"`
	Message  string    `json:"message"`
	Details  string    `json:"details,omitempty"`
	Cause    error     `json:"-"`
	safeWrap *SafeError
}

// SafeError wraps an underlying error with additional info. It implements
// Unwrap so that errors.Is/errors.As can traverse through it to the cause.
type SafeError struct {
	wrapped error
	label   string
}

// NewSafeError creates a new SafeError wrapping the given error.
func NewSafeError(wrapped error, label string) *SafeError {
	return &SafeError{
		wrapped: wrapped,
		label:   label,
	}
}

// Error returns the error string representation.
func (se *SafeError) Error() string {
	if se.label != "" {
		return fmt.Sprintf("%s: %s", se.label, se.wrapped.Error())
	}
	return se.wrapped.Error()
}

// Unwrap returns the wrapped error so the error chain stays intact for
// errors.Is and errors.As.
func (se *SafeError) Unwrap() error {
	return se.wrapped
}

// NewDetailedError creates a new DetailedError.
func NewDetailedError(errType ErrorType, code int, message string) *DetailedError {
	return &DetailedError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
}

// NewDetailedErrorWithCause creates a new DetailedError wrapping a cause.
func NewDetailedErrorWithCause(errType ErrorType, code int, message string, cause error) *DetailedError {
	de := &DetailedError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
	if cause != nil {
		de.Cause = cause
	}
	return de
}

// WrapDetailedError wraps an existing error with additional details.
func WrapDetailedError(errType ErrorType, code int, message string, cause error) *DetailedError {
	de := &DetailedError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
	if cause != nil {
		de.safeWrap = NewSafeError(cause, fmt.Sprintf("[%s:%d] %s", errType, code, message))
		de.Cause = de.safeWrap
	}
	return de
}

// WrapDetailedErrorWithSafe wraps an error and stores it in SafeError.
func WrapDetailedErrorWithSafe(errType ErrorType, code int, message string, cause error) *DetailedError {
	de := &DetailedError{
		Type:    errType,
		Code:    code,
		Message: message,
	}
	if cause != nil {
		de.safeWrap = NewSafeError(cause, fmt.Sprintf("[%s:%d] %s", errType, code, message))
		de.Cause = de.safeWrap
	}
	return de
}

// Error returns the error message.
func (e *DetailedError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s:%d] %s: %v", e.Type, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s:%d] %s", e.Type, e.Code, e.Message)
}

// Unwrap returns the underlying error so errors.Is/errors.As can traverse
// the full chain down to the original cause (including through SafeError).
func (e *DetailedError) Unwrap() error {
	if e.Cause != nil {
		return e.Cause
	}
	return e.safeWrap
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

// WrapSafe wraps an error using SafeError layer.
func (f *ErrFactory) WrapSafe(code int, message string, cause error) *DetailedError {
	return WrapDetailedErrorWithSafe(f.ErrType, code, message, cause)
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

// IsType checks if an error is of a specific type.
func IsType(err error, errType ErrorType) bool {
	if de, ok := err.(*DetailedError); ok {
		return de.Type == errType
	}
	return false
}
