package logalert_test

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Scheduler.ScanInterval = 30 * time.Second
	cfg.Scheduler.EnableAutoScan = true

	logWriter := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelError, logWriter)

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)

	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, cfg, log)

	scheduler := service.NewScheduler(ruleSvc, alertSvc, logStore, cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	var wg sync.WaitGroup

	err := scheduler.Start(ctx, &wg)
	if err != nil {
		t.Fatalf("failed to start scheduler: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	scheduler.Stop()

	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	case <-time.After(3 * time.Second):
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("WaitGroup did not complete after scheduler stop - goroutine leak detected")
	}

	logStore.Close()
	ruleStore.Close()
	alertStore.Close()
}
