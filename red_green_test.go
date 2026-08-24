package logalert

import (
	"context"
	"fmt"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelInfo, logger.NewDiscardWriter())

	logStore := store.NewMemoryLogStore(10000, log)

	now := time.Now()
	for i := 0; i < 5; i++ {
		entry := model.NewLogEntry(fmt.Sprintf("service-%d", i), model.LevelError, "error message")
		entry.Timestamp = now.Add(-5 * time.Minute)
		_ = logStore.Store(context.Background(), entry)
	}

	processingDelay := 100 * time.Millisecond
	logStore.SetProcessingDelay(processingDelay)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel()

	filter := &model.LogFilter{}
	count, err := logStore.Count(ctx, filter)

	hasDefect := false
	if err != nil {
		t.Logf("Count correctly returned error: %v", err)
	} else {
		hasDefect = true
		t.Logf("Count should have returned an error due to context cancellation, but returned count=%d with no error", count)
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Millisecond)
	defer cancel2()

	_, err2 := logStore.Query(ctx2, filter, 100, 0)

	if err2 != nil {
		t.Logf("Query correctly returned error: %v", err2)
	} else {
		hasDefect = true
		t.Log("Query should have returned an error due to context cancellation, but returned results with no error")
	}

	if hasDefect {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Error("缺陷存在：context取消信号未传播到存储层操作，操作在context超时后仍继续执行")
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
