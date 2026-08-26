// Package validator provides input validation utilities.
package validator

import (
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Validator provides validation methods for request parameters.
type Validator struct {
	errors []ValidationError
}

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string
	Message string
}

// Error returns the validation error message.
func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// New creates a new Validator instance.
func New() *Validator {
	return &Validator{
		errors: make([]ValidationError, 0),
	}
}

// HasErrors checks if any validation errors have been recorded.
func (v *Validator) HasErrors() bool {
	return len(v.errors) > 0
}

// Errors returns all validation errors.
func (v *Validator) Errors() []ValidationError {
	return v.errors
}

// ErrorMessage returns a combined error message.
func (v *Validator) ErrorMessage() string {
	if len(v.errors) == 0 {
		return ""
	}
	var msgs []string
	for _, e := range v.errors {
		msgs = append(msgs, e.Error())
	}
	return strings.Join(msgs, "; ")
}

// Require checks that a value is not empty.
func (v *Validator) Require(field, value string) *Validator {
	if strings.TrimSpace(value) == "" {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "this field is required",
		})
	}
	return v
}

// MinLength checks that a string has at least min characters.
func (v *Validator) MinLength(field, value string, min int) *Validator {
	if len(value) < min {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d characters", min),
		})
	}
	return v
}

// MaxLength checks that a string has at most max characters.
func (v *Validator) MaxLength(field, value string, max int) *Validator {
	if len(value) > max {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d characters", max),
		})
	}
	return v
}

// InList checks that a value is in the allowed list.
func (v *Validator) InList(field, value string, allowedValues []string) *Validator {
	for _, allowed := range allowedValues {
		if value == allowed {
			return v
		}
	}
	v.errors = append(v.errors, ValidationError{
		Field:   field,
		Message: fmt.Sprintf("must be one of: %s", strings.Join(allowedValues, ", ")),
	})
	return v
}

// IsInt checks that a string can be parsed as an integer.
func (v *Validator) IsInt(field, value string) *Validator {
	if _, err := strconv.ParseInt(value, 10, 64); err != nil {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must be a valid integer",
		})
	}
	return v
}

// MinInt checks that an integer value is at least min.
func (v *Validator) MinInt(field string, value, min int64) *Validator {
	if value < min {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at least %d", min),
		})
	}
	return v
}

// MaxInt checks that an integer value is at most max.
func (v *Validator) MaxInt(field string, value, max int64) *Validator {
	if value > max {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: fmt.Sprintf("must be at most %d", max),
		})
	}
	return v
}

// IsURL checks that a string is a valid URL.
func (v *Validator) IsURL(field, value string) *Validator {
	if _, err := url.ParseRequestURI(value); err != nil {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must be a valid URL",
		})
	}
	return v
}

// IsEmail checks that a string is a valid email address.
var emailRegex = regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)

func (v *Validator) IsEmail(field, value string) *Validator {
	if !emailRegex.MatchString(value) {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must be a valid email address",
		})
	}
	return v
}

// IsTimestamp checks that a string is a valid timestamp.
func (v *Validator) IsTimestamp(field, value string) *Validator {
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}
	valid := false
	for _, layout := range layouts {
		if _, err := time.Parse(layout, value); err == nil {
			valid = true
			break
		}
	}
	if !valid {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must be a valid timestamp",
		})
	}
	return v
}

// Positive checks that a duration string is positive.
func (v *Validator) Positive(field string, d time.Duration) *Validator {
	if d <= 0 {
		v.errors = append(v.errors, ValidationError{
			Field:   field,
			Message: "must be positive",
		})
	}
	return v
}

// HTTPError returns an appropriate HTTP error response if validation fails.
func (v *Validator) HTTPError(w http.ResponseWriter, statusCode int) bool {
	if !v.HasErrors() {
		return false
	}
	http.Error(w, v.ErrorMessage(), statusCode)
	return true
}
