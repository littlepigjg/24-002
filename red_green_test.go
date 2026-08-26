package main

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Scheduler.ScanInterval = 10 * time.Millisecond

	logWriter := logger.NewDiscardWriter()
	log := logger.NewLogger(logger.LogLevelError, logWriter)

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)

	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, cfg, log)

	scheduler := service.NewScheduler(ruleSvc, alertSvc, logStore, cfg, log)
	cleanupSvc := service.NewCleanupService(logStore, alertStore, cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := scheduler.Start(ctx); err != nil {
		fmt.Printf("RED (红灯，缺陷未修复)\n")
		fmt.Printf("scheduler start failed: %v\n", err)
		os.Exit(1)
	}

	if err := cleanupSvc.Start(ctx); err != nil {
		fmt.Printf("RED (红灯，缺陷未修复)\n")
		fmt.Printf("cleanup service start failed: %v\n", err)
		os.Exit(1)
	}

	time.Sleep(50 * time.Millisecond)

	scheduler.ScanOnce(ctx)
	cleanupSvc.CleanupOnce(ctx)

	time.Sleep(50 * time.Millisecond)

	scheduler.Stop()
	cleanupSvc.Stop()

	time.Sleep(100 * time.Millisecond)

	schedulerIdle := scheduler.AwaitIdle(500 * time.Millisecond)
	cleanupIdle := cleanupSvc.AwaitIdle(500 * time.Millisecond)

	if !schedulerIdle || !cleanupIdle {
		fmt.Printf("RED (红灯，缺陷未修复)\n")
		fmt.Printf("scheduler idle: %v, cleanup idle: %v\n", schedulerIdle, cleanupIdle)
		os.Exit(1)
	}

	fmt.Printf("GREEN (绿灯，缺陷已修复)\n")
	fmt.Printf("scheduler idle: %v, cleanup idle: %v\n", schedulerIdle, cleanupIdle)
}