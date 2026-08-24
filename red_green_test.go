package logalert_test

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	var issueFound int32
	var resultPrinted int32

	defer func() {
		if atomic.LoadInt32(&resultPrinted) == 0 {
			if atomic.LoadInt32(&issueFound) == 1 {
				fmt.Println("RED（红灯，缺陷未修复）")
			} else {
				fmt.Println("GREEN（绿灯，缺陷已修复）")
			}
		}
	}()

	logStore := store.NewMemoryLogStore(100, logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter()))
	ctx := context.Background()

	for i := 0; i < 100; i++ {
		entry := model.NewLogEntry("svc", model.LevelError, "error message")
		logStore.Store(ctx, entry)
	}

	initSnap := logStore.RawSnapshot()
	if len(initSnap) == 0 {
		t.Fatal("store initialization failed: entries should not be empty")
	}

	var storeMu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.StoreInt32(&issueFound, 1)
				}
			}()
			storeMu.Lock()
			entry := model.NewLogEntry("svc2", model.LevelWarn, "concurrent write")
			logStore.Store(ctx, entry)
			storeMu.Unlock()
		}()
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if r := recover(); r != nil {
					atomic.StoreInt32(&issueFound, 1)
				}
			}()
			for j := 0; j < 200; j++ {
				snap := logStore.RawSnapshot()
				_ = len(snap)
			}
		}()
	}

	wg.Wait()

	finalSnap := logStore.RawSnapshot()
	finalSize := len(finalSnap)

	if finalSize < 30 {
		atomic.StoreInt32(&issueFound, 1)
	}

	if atomic.LoadInt32(&issueFound) == 1 {
		fmt.Println("RED（红灯，缺陷未修复）")
		atomic.StoreInt32(&resultPrinted, 1)
		t.Errorf("data integrity violation or race condition detected: final size=%d", finalSize)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
		atomic.StoreInt32(&resultPrinted, 1)
	}
}
