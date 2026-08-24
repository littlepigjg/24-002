package main

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"logalert/pkg/cache"
	"logalert/pkg/errors"
	"logalert/pkg/retry"
)

func TestRedGreen(t *testing.T) {
	ctx := context.Background()

	// RED 1: not_found 错误经缓存 GetOrSet 包装后被错误重试
	t.Run("not-found error via GetOrSet should NOT be retried", func(t *testing.T) {
		lc := cache.NewLazy(100)
		notFoundErr := errors.NotFoundErrorFactory.New(404, "resource not found")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 10 * time.Millisecond
		retryCfg.MaxDelay = 10 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			_, loadErr := lc.GetOrSet("key1", func() (interface{}, error) {
				return nil, notFoundErr
			}, 0)
			return loadErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected error but got nil")
			return
		}
		if callCount > 1 {
			t.Errorf("RED (红灯，缺陷未修复): not-found error was retried %d times through cache GetOrSet, should have failed on first attempt", callCount)
			return
		}
		if !strings.Contains(err.Error(), "resource not found") {
			t.Errorf("RED (红灯，缺陷未修复): error lost original message, got: %s", err.Error())
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): not-found error correctly stopped on first attempt after cache wrapping")
	})

	// RED 2: not_found 错误经 LoadOrCompute 路径同样被错误重试
	t.Run("not-found error via LoadOrCompute should NOT be retried", func(t *testing.T) {
		lc := cache.NewLazy(100)
		notFoundErr := errors.NotFoundErrorFactory.New(404, "not found via compute")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 10 * time.Millisecond
		retryCfg.MaxDelay = 10 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			_, loadErr := lc.LoadOrCompute("key2", func() (interface{}, error) {
				return nil, notFoundErr
			}, 0)
			return loadErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected error but got nil")
			return
		}
		if callCount > 1 {
			t.Errorf("RED (红灯，缺陷未修复): not-found error was retried %d times through LoadOrCompute, should have failed on first attempt", callCount)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): not-found error correctly stopped on first attempt via LoadOrCompute")
	})

	// RED 3: validation 类型错误经缓存包装后同样被错误重试
	t.Run("validation error via GetOrSet should NOT be retried", func(t *testing.T) {
		lc := cache.NewLazy(100)
		validationErr := errors.ValidationErrorFactory.New(400, "invalid input")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 10 * time.Millisecond
		retryCfg.MaxDelay = 10 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			_, loadErr := lc.GetOrSet("key3", func() (interface{}, error) {
				return nil, validationErr
			}, 0)
			return loadErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected error but got nil")
			return
		}
		if callCount > 1 {
			t.Errorf("RED (红灯，缺陷未修复): validation error was retried %d times through cache, should have failed on first attempt", callCount)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): validation error correctly stopped on first attempt after cache wrapping")
	})

	// RED 4: conflict 类型错误经缓存包装后同样被错误重试
	t.Run("conflict error via GetOrSet should NOT be retried", func(t *testing.T) {
		lc := cache.NewLazy(100)
		conflictErr := errors.NewDetailedError(errors.ErrorTypeConflict, 409, "duplicate entry")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 10 * time.Millisecond
		retryCfg.MaxDelay = 10 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			_, loadErr := lc.GetOrSet("key4", func() (interface{}, error) {
				return nil, conflictErr
			}, 0)
			return loadErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected error but got nil")
			return
		}
		if callCount > 1 {
			t.Errorf("RED (红灯，缺陷未修复): conflict error was retried %d times through cache, should have failed on first attempt", callCount)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): conflict error correctly stopped on first attempt after cache wrapping")
	})

	// RED 5: retry.DoWithClassification 对缓存包装后的错误分类错误
	t.Run("not-found error misclassified by DoWithClassification after cache wrapping", func(t *testing.T) {
		lc := cache.NewLazy(100)
		notFoundErr := errors.NotFoundErrorFactory.New(404, "classification test")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 1

		_, errType, isRetryable, category := retry.DoWithClassification(ctx, retryCfg, func() error {
			_, loadErr := lc.GetOrSet("key5", func() (interface{}, error) {
				return nil, notFoundErr
			}, 0)
			return loadErr
		})

		if errType != errors.ErrorTypeNotFound {
			t.Errorf("RED (红灯，缺陷未修复): error type not preserved after cache wrapping, expected 'not_found' but got '%s'. category=%s, retryable=%v", errType, category, isRetryable)
			return
		}
		if isRetryable {
			t.Errorf("RED (红灯，缺陷未修复): not-found error classified as retryable after cache wrapping, should be non-retryable. category=%s", category)
			return
		}
		if category != "permanent" {
			t.Errorf("RED (红灯，缺陷未修复): not-found error category should be 'permanent' but got '%s'", category)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): not-found error correctly classified as permanent non-retryable after cache wrapping")
	})

	// RED 6: context 类型错误经缓存包装后被错误重试
	t.Run("context error via GetOrSet should NOT be retried", func(t *testing.T) {
		lc := cache.NewLazy(100)
		contextErr := errors.NewDetailedError(errors.ErrorTypeContext, 408, "context deadline exceeded")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 10 * time.Millisecond
		retryCfg.MaxDelay = 10 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			_, loadErr := lc.GetOrSet("key6", func() (interface{}, error) {
				return nil, contextErr
			}, 0)
			return loadErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected error but got nil")
			return
		}
		if callCount > 1 {
			t.Errorf("RED (红灯，缺陷未修复): context error was retried %d times through cache, should have failed on first attempt", callCount)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): context error correctly stopped on first attempt after cache wrapping")
	})

	// GREEN 1: 直接传递 DetailedError 不经缓存时行为正确（基线验证）
	t.Run("direct not-found DetailedError should NOT be retried", func(t *testing.T) {
		notFoundErr := errors.NotFoundErrorFactory.New(404, "item missing")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 5 * time.Millisecond
		retryCfg.MaxDelay = 5 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			return notFoundErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected error but got nil")
			return
		}
		if callCount > 1 {
			t.Errorf("RED (红灯，缺陷未修复): direct DetailedError was retried %d times, should have failed immediately", callCount)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): direct DetailedError correctly identified without wrapping")
	})

	// GREEN 2: database 类型错误经缓存包装后应继续重试（基线验证）
	t.Run("database error via GetOrSet should be retried", func(t *testing.T) {
		lc := cache.NewLazy(100)
		dbErr := errors.DatabaseErrorFactory.New(5001, "connection timeout")
		retryCfg := retry.DefaultConfig()
		retryCfg.MaxAttempts = 3
		retryCfg.InitialDelay = 10 * time.Millisecond
		retryCfg.MaxDelay = 10 * time.Millisecond

		callCount := 0
		err := retry.Do(ctx, retryCfg, func() error {
			callCount++
			_, loadErr := lc.GetOrSet("key7", func() (interface{}, error) {
				return nil, dbErr
			}, 0)
			return loadErr
		})

		if err == nil {
			t.Errorf("RED (红灯，缺陷未修复): expected retry error but got nil")
			return
		}
		if callCount < 2 {
			t.Errorf("RED (红灯，缺陷未修复): database error was only called %d times, should have been retried at least twice", callCount)
			return
		}
		t.Log("GREEN (绿灯，缺陷已修复): database error correctly retried after cache wrapping")
	})

	fmt.Println("All tests completed")
}