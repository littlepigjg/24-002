package main

import (
	"fmt"
	stdErrors "errors"
	"testing"

	appErrors "logalert/pkg/errors"
)

func TestRedGreen(t *testing.T) {
	allPassed := true

	// Test 1: errors.Is on DetailedError wrapping AppError
	t.Run("Test 1: errors.Is on DetailedError wrapping AppError", func(t *testing.T) {
		innerErr := appErrors.ErrNotFound
		wrappedErr := appErrors.NotFoundErrorFactory.Wrap(4001, "not found in store", innerErr)
		if stdErrors.Is(wrappedErr, innerErr) {
			fmt.Println("PASS: errors.Is correctly finds the inner ErrNotFound through DetailedError")
		} else {
			fmt.Println("FAIL: errors.Is fails to find inner ErrNotFound through DetailedError wrapping (SafeError broke the chain)")
			t.Errorf("errors.Is failed to find inner ErrNotFound through DetailedError wrapping")
			allPassed = false
		}
	})

	// Test 2: errors.Is on AppError wrapping another AppError
	t.Run("Test 2: errors.Is on AppError wrapping another AppError", func(t *testing.T) {
		originalErr := appErrors.New(5001, "database connection failed")
		wrappedAppErr := appErrors.Wrap(5002, "failed to process request", originalErr)
		if stdErrors.Is(wrappedAppErr, originalErr) {
			fmt.Println("PASS: errors.Is correctly finds the original error through AppError wrapping")
		} else {
			fmt.Println("FAIL: errors.Is fails to find original error through AppError wrapping (SafeError broke the chain)")
			t.Errorf("errors.Is failed to find original error through AppError wrapping")
			allPassed = false
		}
	})

	// Test 3: errors.Is through multiple layers of wrapping
	t.Run("Test 3: errors.Is through multiple layers of wrapping", func(t *testing.T) {
		baseErr := appErrors.ErrInvalidInput
		level1 := appErrors.NewWithCause(1002, "invalid user input", baseErr)
		level2 := appErrors.ValidationErrorFactory.Wrap(1003, "validation failed in service", level1)
		if stdErrors.Is(level2, baseErr) {
			fmt.Println("PASS: errors.Is correctly traverses multiple wrapping layers")
		} else {
			fmt.Println("FAIL: errors.Is fails to traverse multiple wrapping layers (SafeError breaks each layer)")
			t.Errorf("errors.Is failed to traverse multiple wrapping layers")
			allPassed = false
		}
	})

	// Test 4: errors.As on DetailedError wrapping AppError
	t.Run("Test 4: errors.As on DetailedError wrapping AppError", func(t *testing.T) {
		innerErr := appErrors.ErrNotFound
		wrappedErr := appErrors.NotFoundErrorFactory.Wrap(4001, "not found in store", innerErr)
		var target *appErrors.AppError
		if stdErrors.As(wrappedErr, &target) {
			fmt.Println("PASS: errors.As correctly finds *AppError through DetailedError wrapping")
		} else {
			fmt.Println("FAIL: errors.As fails to find *AppError through DetailedError wrapping")
			t.Errorf("errors.As failed to find *AppError through DetailedError wrapping")
			allPassed = false
		}
	})

	// Test 5: Custom Is checks using the package's helper functions
	t.Run("Test 5: IsNotFound helper after SafeError wrapping", func(t *testing.T) {
		notFoundErr := appErrors.NotFound("user-123")
		if appErrors.IsNotFound(notFoundErr) {
			fmt.Println("PASS: IsNotFound correctly identifies a direct AppError")
		} else {
			fmt.Println("FAIL: IsNotFound fails on a direct AppError (should not happen)")
			t.Errorf("IsNotFound failed on a direct AppError")
			allPassed = false
		}

		wrappedNotFound := appErrors.NotFoundErrorFactory.Wrap(4001, "not found in store", notFoundErr)
		if appErrors.IsNotFound(wrappedNotFound) {
			fmt.Println("PASS: IsNotFound correctly identifies not-found error through DetailedError wrapping")
		} else {
			fmt.Println("FAIL: IsNotFound fails to identify not-found error through DetailedError wrapping")
			t.Errorf("IsNotFound failed to identify not-found error through DetailedError wrapping")
			allPassed = false
		}
	})

	// Summary
	fmt.Println("\n========================================")
	if allPassed {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Fail()
	}
	fmt.Println("========================================")
}