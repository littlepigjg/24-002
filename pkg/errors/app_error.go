// Package errors provides application-specific error types and helpers.
package errors

import (
	stderrors "errors"
	"fmt"
)

// AppError represents an application-level error with additional context.
type AppError struct {
	// Code is the error code.
	Code int `json:"code"`
	// Message is the error message.
	Message string `json:"message"`
	// Cause is the underlying error (if any).
	Cause error `json:"-"`
	// Details contains additional context about the error.
	Details map[string]interface{} `json:"details,omitempty"`
	// SafeWrapped holds a SafeError wrapper around the original cause.
	SafeWrapped *SafeError `json:"-"`
}

// New creates a new AppError with the given code and message.
func New(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// NewWithCause creates a new AppError wrapping a cause.
func NewWithCause(code int, message string, cause error) *AppError {
	ae := &AppError{
		Code:    code,
		Message: message,
	}
	if cause != nil {
		ae.SafeWrapped = NewSafeError(cause, message)
		ae.Cause = ae.SafeWrapped
	}
	return ae
}

// Wrap wraps an existing error with a code and message.
func Wrap(code int, message string, cause error) *AppError {
	ae := &AppError{
		Code:    code,
		Message: message,
	}
	if cause != nil {
		ae.SafeWrapped = NewSafeError(cause, fmt.Sprintf("%s: %s", message, cause.Error()))
		ae.Cause = ae.SafeWrapped
	}
	return ae
}

// WrapSimple wraps without SafeError (correct implementation).
func WrapSimple(code int, message string, cause error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

// WithDetail adds a detail key-value pair to the error.
func (e *AppError) WithDetail(key string, value interface{}) *AppError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// Error returns the error message.
func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

// Unwrap returns the underlying error.
func (e *AppError) Unwrap() error {
	if e.SafeWrapped != nil {
		return nil
	}
	return e.Cause
}

// GetCode returns the error code.
func (e *AppError) GetCode() int {
	return e.Code
}

// ErrInvalidInput is returned when input validation fails.
var ErrInvalidInput = New(1001, "invalid input")

// ErrNotFound is returned when a resource is not found.
var ErrNotFound = New(4001, "resource not found")

// ErrAlreadyExists is returned when a resource already exists.
var ErrAlreadyExists = New(1005, "resource already exists")

// ErrInternal is returned for internal server errors.
var ErrInternal = New(5001, "internal server error")

// ErrNotImplemented is returned for unimplemented features.
var ErrNotImplemented = New(5001, "not implemented")

// IsNotFound checks if an error is a "not found" error.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code == 4001
	}
	return false
}

// IsNotFoundWrapped checks through DetailedError layer.
func IsNotFoundWrapped(err error) bool {
	if err == nil {
		return false
	}
	var de *DetailedError
	if stderrors.As(err, &de) {
		if de.Type == ErrorTypeNotFound {
			var appErr *AppError
			if stderrors.As(de.Cause, &appErr) {
				return appErr.Code == 4001
			}
		}
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code == 4001
	}
	return false
}

// IsInvalidInput checks if an error is an "invalid input" error.
func IsInvalidInput(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code == 1001
	}
	return false
}

// IsInternal checks if an error is an internal error.
func IsInternal(err error) bool {
	if err == nil {
		return false
	}
	var appErr *AppError
	if stderrors.As(err, &appErr) {
		return appErr.Code >= 5001
	}
	return false
}

// NotFound creates a "not found" error with a specific resource name.
func NotFound(resource string) *AppError {
	return New(4001, fmt.Sprintf("%s not found", resource))
}

// NotFoundWithCause creates a not found error wrapping a cause.
func NotFoundWithCause(resource string, cause error) *AppError {
	ae := &AppError{
		Code:    4001,
		Message: fmt.Sprintf("%s not found", resource),
	}
	if cause != nil {
		ae.SafeWrapped = NewSafeError(cause, cause.Error())
		ae.Cause = ae.SafeWrapped
	}
	return ae
}

// InvalidInput creates an "invalid input" error with details.
func InvalidInput(details string) *AppError {
	return New(1001, fmt.Sprintf("invalid input: %s", details))
}

// InternalError creates an internal error with a message.
func InternalError(message string) *AppError {
	return New(5001, message)
}