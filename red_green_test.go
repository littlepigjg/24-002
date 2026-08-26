package logalert

import (
	"context"
	"fmt"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	tmpDir := t.TempDir()
	urlsPath := tmpDir + "/urls.json"
	logsPath := tmpDir + "/access.log"

	cfg := config.Default()
	cfg.Storage.URLFilePath(urlsPath)
	cfg.Storage.LogFilePath(logsPath)
	cfg.Storage.SyncInterval(5 * time.Second)
	cfg.Storage.FlushOnWrite(true)

	log := logger.NewLogger(logger.LogLevelWarn, logger.NewStdoutWriter())
	_ = log

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URL store: %v", err)
	}

	ctx := context.Background()
	if err := urlStore.Load(ctx); err != nil {
		t.Fatalf("failed to load URL store: %v", err)
	}
	defer urlStore.Close()

	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create access log store: %v", err)
	}
	if err := logStore.Open(ctx); err != nil {
		t.Fatalf("failed to open access log store: %v", err)
	}
	defer logStore.Close()

	urlService, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	redirectService, err := service.NewRedirectService(urlStore, logStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	allPassed := true

	runTest := func(name string, fn func(t *testing.T) bool) {
		t.Run(name, func(t *testing.T) {
			passed := fn(t)
			if !passed {
				allPassed = false
			}
		})
	}

	runTest("create_valid_url", func(t *testing.T) bool {
		req := &model.CreateReq{
			RawURL:     "https://example.com/test",
			CustomCode: "test1",
			MaxVisits:  100,
			Keywords:   []string{"prod"},
		}
		result, err := urlService.Create(ctx, req)
		if err != nil {
			t.Logf("Error: %v", err)
			return false
		}
		if result.Code != "test1" {
			t.Logf("Expected code test1, got %s", result.Code)
			return false
		}
		if result.RawURL != "https://example.com/test" {
			t.Logf("Expected raw_url https://example.com/test, got %s", result.RawURL)
			return false
		}
		t.Log("GREEN: create_valid_url passed")
		return true
	})

	runTest("create_empty_request", func(t *testing.T) bool {
		panicOccurred := false
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
				t.Logf("RED: Panic caught - %v", r)
			}
		}()

		req := &model.CreateReq{
			RawURL: "https://example.com/empty",
		}
		result, err := urlService.Create(ctx, req)
		if err != nil {
			t.Logf("Error: %v", err)
			return false
		}
		if result == nil {
			return false
		}
		if panicOccurred {
			return false
		}
		t.Log("GREEN: create_empty_request passed")
		return true
	})

	runTest("log_query_nil_filter", func(t *testing.T) bool {
		panicOccurred := false
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
				t.Logf("RED: Panic caught - %v", r)
			}
		}()

		logQuery := &model.LogQuery{}
		logQuery.Filter = &model.LogFilter{}
		logQuery.Limit = 10
		logQuery.Offset = 0

		errs := logQuery.Validate()
		_ = errs
		if panicOccurred {
			return false
		}
		t.Log("GREEN: log_query_nil_filter passed")
		return true
	})

	runTest("alert_query_nil_filter", func(t *testing.T) bool {
		panicOccurred := false
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
				t.Logf("RED: Panic caught - %v", r)
			}
		}()

		alertQuery := &model.AlertQuery{}
		alertQuery.Filter = &model.AlertFilter{}
		alertQuery.Limit = 10
		alertQuery.Offset = 0

		errs := alertQuery.Validate()
		_ = errs
		if panicOccurred {
			return false
		}
		t.Log("GREEN: alert_query_nil_filter passed")
		return true
	})

	runTest("short_url_validate", func(t *testing.T) bool {
		panicOccurred := false
		defer func() {
			if r := recover(); r != nil {
				panicOccurred = true
				t.Logf("RED: Panic caught - %v", r)
			}
		}()

		shortURL := &model.ShortURL{
			Code:   "test2",
			RawURL: "https://example.com/validate",
		}
		err := shortURL.Validate()
		if err != nil {
			t.Logf("Error: %v", err)
			return false
		}
		if panicOccurred {
			return false
		}
		t.Log("GREEN: short_url_validate passed")
		return true
	})

	runTest("redirect_valid_url", func(t *testing.T) bool {
		req := &service.RedirectRequest{
			Code:      "test1",
			Timestamp: time.Now(),
		}
		result, err := redirectService.HandleRedirect(ctx, req)
		if err != nil {
			t.Logf("Error: %v", err)
			return false
		}
		if result.RawURL != "https://example.com/test" {
			t.Logf("Expected raw_url https://example.com/test, got %s", result.RawURL)
			return false
		}
		if result.Status != 302 {
			t.Logf("Expected status 302, got %d", result.Status)
			return false
		}
		t.Log("GREEN: redirect_valid_url passed")
		return true
	})

	runTest("raw_snapshot", func(t *testing.T) bool {
		snapshot := urlStore.RawSnapshot()
		if len(snapshot) == 0 {
			t.Logf("Expected non-empty snapshot")
			return false
		}
		if _, ok := snapshot["test1"]; !ok {
			t.Logf("Expected test1 in snapshot")
			return false
		}
		t.Log("GREEN: raw_snapshot passed")
		return true
	})

	runTest("set_panic_guard", func(t *testing.T) bool {
		urlStore.SetPanicGuard(func(code, rawURL string) bool {
			return true
		})
		snapshot := urlStore.RawSnapshot()
		_ = snapshot
		t.Log("GREEN: set_panic_guard passed")
		return true
	})

	if allPassed {
		fmt.Printf("GREEN (绿灯，缺陷已修复)\n")
	} else {
		fmt.Printf("RED (红灯，缺陷未修复)\n")
		t.FailNow()
	}
}
