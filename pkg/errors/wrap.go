// Package errors provides application-specific error types and helpers.
package errors

import (
	"errors"
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
	// ErrorTypeLimitExceeded is for rate limit / capacity errors.
	ErrorTypeLimitExceeded ErrorType = "limit_exceeded"
	// ErrorTypeStateConflict is for invalid state transitions.
	ErrorTypeStateConflict ErrorType = "state_conflict"
)

// ServiceError represents a service-layer error with a typed kind and error code.
type ServiceError struct {
	Kind    string
	Code    int
	Message string
	Cause   error
}

// NewServiceError creates a new ServiceError.
func NewServiceError(kind string, code int, message string) *ServiceError {
	return &ServiceError{
		Kind:    kind,
		Code:    code,
		Message: message,
	}
}

// WrapServiceError creates a ServiceError wrapping a cause error.
func WrapServiceError(kind string, code int, message string, cause error) *ServiceError {
	return &ServiceError{
		Kind:    kind,
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// Error returns the error string representation.
func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying cause error.
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// IsKind checks if an error is a ServiceError with the given kind using errors.As.
func IsKind(err error, kind string) bool {
	if err == nil {
		return false
	}
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr.Kind == kind
	}
	return false
}

// GetCode extracts the error code from a ServiceError, or returns 0 if not a ServiceError.
func GetCode(err error) int {
	if err == nil {
		return 0
	}
	var svcErr *ServiceError
	if errors.As(err, &svcErr) {
		return svcErr.Code
	}
	return 0
}

// ErrKind constants for ServiceError.Kind values.
const (
	ErrKindNotFound       = "not_found"
	ErrKindValidation     = "validation"
	ErrKindLimitExceeded  = "limit_exceeded"
	ErrKindStateConflict  = "state_conflict"
	ErrKindInternal       = "internal"
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

// IsType checks if an error is of a specific type.
func IsType(err error, errType ErrorType) bool {
	if de, ok := err.(*DetailedError); ok {
		return de.Type == errType
	}
	return false
}
