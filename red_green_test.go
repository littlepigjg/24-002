package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	logWriter := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWriter)

	tmpDir, err := os.MkdirTemp("", "logalert-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	fp, err := store.NewFilePersistence(tmpDir, log)
	if err != nil {
		t.Fatalf("failed to create file persistence: %v", err)
	}

	fp.SetFaultInjector(func() error {
		return fmt.Errorf("simulated disk write failure")
	})

	logStore := store.NewMemoryLogStore(1000, log)
	alertStore := store.NewMemoryAlertStore(1000, log)

	now := time.Now()
	oldEntry := model.NewLogEntry("service-a", model.LevelError, "old error log")
	oldEntry.Timestamp = now.Add(-10 * 24 * time.Hour)
	oldEntry.ReceivedAt = now.Add(-10 * 24 * time.Hour)

	newEntry := model.NewLogEntry("service-b", model.LevelInfo, "new info log")
	newEntry.Timestamp = now
	newEntry.ReceivedAt = now

	if err := logStore.Store(context.Background(), oldEntry); err != nil {
		t.Fatalf("failed to store old entry: %v", err)
	}
	if err := logStore.Store(context.Background(), newEntry); err != nil {
		t.Fatalf("failed to store new entry: %v", err)
	}

	cfg := config.DefaultConfig()
	cleanupSvc := service.NewCleanupService(logStore, alertStore, cfg, log)
	cleanupSvc.SetFileStore(fp)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err = cleanupSvc.CleanupOnce(ctx)

	if err == nil {
		t.Log("RED (红灯，缺陷未修复)")
		t.Log("CleanupOnce 返回 nil，未正确传播 SaveState 故障注入器返回的错误。")
		t.Log("当 SaveState 因故障注入而失败时，CleanupOnce 应该返回错误，但实际返回了 nil。")
		t.FailNow()
	}

	logStore2 := store.NewMemoryLogStore(1000, log)
	alertStore2 := store.NewMemoryAlertStore(1000, log)

	fp2, err := store.NewFilePersistence(filepath.Join(tmpDir, "normal"), log)
	if err != nil {
		t.Fatalf("failed to create file persistence 2: %v", err)
	}

	oldEntry2 := model.NewLogEntry("service-a", model.LevelError, "old error log")
	oldEntry2.Timestamp = now.Add(-10 * 24 * time.Hour)
	oldEntry2.ReceivedAt = now.Add(-10 * 24 * time.Hour)
	newEntry2 := model.NewLogEntry("service-b", model.LevelInfo, "new info log")

	if err := logStore2.Store(context.Background(), oldEntry2); err != nil {
		t.Fatalf("failed to store old entry 2: %v", err)
	}
	if err := logStore2.Store(context.Background(), newEntry2); err != nil {
		t.Fatalf("failed to store new entry 2: %v", err)
	}

	cleanupSvc2 := service.NewCleanupService(logStore2, alertStore2, cfg, log)
	cleanupSvc2.SetFileStore(fp2)

	err2 := cleanupSvc2.CleanupOnce(ctx)
	if err2 != nil {
		t.Logf("CleanupOnce with normal persistence returned error: %v", err2)
	}

	logsAfter, _ := logStore2.Query(ctx, nil, 100000000, 0)
	if len(logsAfter) != 1 {
		t.Logf("RED (红灯，缺陷未修复)")
		t.Logf("清理后应该只剩1条新日志，实际剩余: %d", len(logsAfter))
		t.FailNow()
	}

	t.Log("GREEN (绿灯，缺陷已修复)")
	t.Log("CleanupOnce 正确传播了 SaveState 的错误，且在无故障注入时能正确清理过期数据。")
}
