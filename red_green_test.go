package main

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"

	"logalert/internal/handler"
	"logalert/internal/service"
	"logalert/pkg/logger"
)

type mockScheduler struct{}

func (m *mockScheduler) Start(ctx context.Context) error             { return nil }
func (m *mockScheduler) Stop()                                       {}
func (m *mockScheduler) ScanOnce(ctx context.Context) error          { return nil }
func (m *mockScheduler) GetStatus() service.SchedulerStatus         { return service.SchedulerStatus{} }

func TestRedGreen(t *testing.T) {
	logWriter := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWriter)

	metricStore := handler.NewHealthMetricStore()
	metricStore.ClearMetrics()

	healthHandler := handler.NewHealthHandlerWithStore(log, metricStore)
	schedulerHandler := handler.NewSchedulerHandler(&mockScheduler{}, log, metricStore)

	test1Panic := false
	test2Panic := false

	// Test 1: HandleHealth with empty metrics - should not panic when fixed
	func() {
		defer func() {
			if r := recover(); r != nil {
				test1Panic = true
			}
		}()

		req := httptest.NewRequest("GET", "/health", nil)
		rec := httptest.NewRecorder()
		healthHandler.HandleHealth(rec, req)
	}()

	if test1Panic {
		t.Error("RED（红灯，缺陷未修复）- HandleHealth 发生 panic，空切片未检查")
	} else {
		t.Log("GREEN（绿灯，缺陷已修复）- HandleHealth 正常处理空切片")
	}

	// Test 2: GetStatus with empty metrics - should not panic when fixed
	func() {
		defer func() {
			if r := recover(); r != nil {
				test2Panic = true
			}
		}()

		req := httptest.NewRequest("GET", "/api/scheduler/status", nil)
		rec := httptest.NewRecorder()
		schedulerHandler.GetStatus(rec, req)
	}()

	if test2Panic {
		t.Error("RED（红灯，缺陷未修复）- GetStatus 发生 panic，空切片未检查")
	} else {
		t.Log("GREEN（绿灯，缺陷已修复）- GetStatus 正常处理空切片")
	}

	if test1Panic || test2Panic {
		fmt.Println("最终判定：RED（红灯，缺陷未修复）")
	} else {
		fmt.Println("最终判定：GREEN（绿灯，缺陷已修复）")
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
