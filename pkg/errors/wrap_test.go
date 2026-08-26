package errors

import (
	stderrors "errors"
	"fmt"
	"testing"
)

func TestAppErrorWrapErrorsIs(t *testing.T) {
	wrapped := Wrap(5001, "lookup failed", ErrNotFound)
	if !stderrors.Is(wrapped, ErrNotFound) {
		t.Errorf("errors.Is(Wrap(ErrNotFound)) = false, want true")
	}
	// IsNotFound relies on errors.As walking the chain.
	if !IsNotFound(wrapped) {
		t.Errorf("IsNotFound(Wrap(ErrNotFound)) = false, want true")
	}
}

func TestMultiLayerWrapErrorsIs(t *testing.T) {
	inner := Wrap(5001, "db error", ErrInvalidInput)
	outer := Wrap(5002, "service error", inner)
	if !stderrors.Is(outer, ErrInvalidInput) {
		t.Errorf("errors.Is(outer, ErrInvalidInput) = false, want true")
	}
	if !IsInvalidInput(outer) {
		t.Errorf("IsInvalidInput(outer) = false, want true")
	}
}

func TestDetailedErrorWrapErrorsIs(t *testing.T) {
	cause := ErrNotFound
	de := NotFoundErrorFactory.Wrap(4040, "missing", cause)
	if !stderrors.Is(de, cause) {
		t.Errorf("errors.Is(DetailedError.Wrap, ErrNotFound) = false, want true")
	}
	// Multi-layer DetailedError.
	inner := NotFoundErrorFactory.Wrap(4041, "inner", cause)
	outer := InternalErrorFactory.Wrap(5003, "outer", inner)
	if !stderrors.Is(outer, cause) {
		t.Errorf("errors.Is(multi-layer DetailedError, ErrNotFound) = false, want true")
	}
}

func TestErrorsAsAppError(t *testing.T) {
	inner := Wrap(5001, "layer1", ErrNotFound)
	outer := fmt.Errorf("outer: %w", inner) // stdlib wrapping on top
	var target *AppError
	if !stderrors.As(outer, &target) {
		t.Fatalf("errors.As(*AppError) = false, want true")
	}
	if target.Code != 5001 {
		t.Errorf("target.Code = %d, want 5001", target.Code)
	}
}

func TestSafeErrorUnwrap(t *testing.T) {
	se := NewSafeError(ErrNotFound, "label")
	if !stderrors.Is(se, ErrNotFound) {
		t.Errorf("errors.Is(SafeError, ErrNotFound) = false, want true")
	}
}
