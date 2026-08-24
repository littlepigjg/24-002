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
)

func TestRedGreen(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "shurl-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	invalidConfigPath := filepath.Join(tmpDir, "invalid.json")
	if err := os.WriteFile(invalidConfigPath, []byte("this is not valid json"), 0644); err != nil {
		t.Fatalf("failed to write invalid config: %v", err)
	}

	_, loadErr := config.Load(invalidConfigPath)
	if loadErr == nil {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Fatalf("expected error from config.Load with invalid config file, got nil")
	}

	cfg := config.Default()
	cfg.Storage.DataDir = tmpDir
	cfg.Storage.URLFilePath(tmpDir)
	cfg.Storage.LogFilePath(tmpDir)
	cfg.Storage.SyncInterval(time.Second)
	cfg.Storage.FlushOnWrite(true)

	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatalf("failed to create URL store: %v", err)
	}

	ctx := context.Background()
	if err := urlStore.Load(ctx); err != nil {
		t.Fatalf("failed to load URL store: %v", err)
	}

	urlSvc, err := service.NewURLService(cfg, urlStore)
	if err != nil {
		t.Fatalf("failed to create URL service: %v", err)
	}

	shortURL, err := urlSvc.Create(ctx, &model.CreateReq{
		RawURL:     "https://example.com/very/long/url/path",
		CustomCode: "testcode",
		MaxVisits:  100,
	})
	if err != nil {
		t.Fatalf("failed to create short URL: %v", err)
	}

	if shortURL.Code != "testcode" {
		t.Errorf("expected code 'testcode', got '%s'", shortURL.Code)
	}

	if shortURL.RawURL != "https://example.com/very/long/url/path" {
		t.Errorf("expected raw URL, got '%s'", shortURL.RawURL)
	}

	if shortURL.Custom != true {
		t.Errorf("expected custom to be true")
	}

	if shortURL.IsExpired(time.Now()) {
		t.Errorf("expected short URL not to be expired")
	}

	err = urlStore.Save(shortURL, true)
	if err != nil {
		t.Fatalf("failed to save short URL: %v", err)
	}

	fetched, err := urlStore.Get("testcode")
	if err != nil {
		t.Fatalf("failed to get short URL: %v", err)
	}

	if fetched.RawURL != "https://example.com/very/long/url/path" {
		t.Errorf("expected raw URL from store, got '%s'", fetched.RawURL)
	}

	urlStore.SetPanicGuard(func(code, rawURL string) bool {
		return false
	})

	snapshot := urlStore.RawSnapshot()
	if len(snapshot) != 1 {
		t.Errorf("expected 1 URL in snapshot, got %d", len(snapshot))
	}

	accessLogStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatalf("failed to create access log store: %v", err)
	}

	if err := accessLogStore.Open(ctx); err != nil {
		t.Fatalf("failed to open access log store: %v", err)
	}

	redirectSvc, err := service.NewRedirectService(urlStore, accessLogStore)
	if err != nil {
		t.Fatalf("failed to create redirect service: %v", err)
	}

	result, err := redirectSvc.HandleRedirect(ctx, &model.RedirectRequest{
		Code:      "testcode",
		Timestamp: time.Now(),
	})
	if err != nil {
		t.Fatalf("failed to handle redirect: %v", err)
	}

	if result.RawURL != "https://example.com/very/long/url/path" {
		t.Errorf("expected raw URL, got '%s'", result.RawURL)
	}
	if result.Status != 302 {
		t.Errorf("expected status 302, got %d", result.Status)
	}

	if err := urlStore.Close(); err != nil {
		t.Fatalf("failed to close URL store: %v", err)
	}

	if err := accessLogStore.Close(); err != nil {
		t.Fatalf("failed to close access log store: %v", err)
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
}
