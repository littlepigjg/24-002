package logalert

import (
	"context"
	"fmt"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/internal/service"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.URLFilePath("./test_urls.json")
	cfg.Storage.LogFilePath("./test_access.json")
	cfg.Storage.SyncInterval(1 * time.Second)
	cfg.Storage.FlushOnWrite(true)

	s, err := store.NewURLStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()

	ls, err := store.NewAccessLogStore(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer ls.Close()

	us, err := service.NewURLService(cfg, s)
	if err != nil {
		t.Fatal(err)
	}

	rs, err := service.NewRedirectService(s, ls)
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if err := s.Load(ctx); err == nil {
		fmt.Println("RED (红灯，缺陷未修复)")
		fmt.Println("缺陷：Load 方法忽略了已取消的 context，继续执行操作")
		fmt.Println("修复后：Load 方法应在检测到 context 取消时立即返回错误")
		t.Errorf("context cancellation not propagated: Load succeeded with cancelled context, expected error")
		return
	}

	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	if err := s.Load(ctx2); err != nil {
		t.Fatalf("Load with valid context should succeed: %v", err)
	}

	snap := s.RawSnapshot()
	if len(snap) != 0 {
		t.Fatalf("snapshot should be empty, got %d entries", len(snap))
	}

	req := &model.CreateReq{
		RawURL:     "http://example.com/test",
		CustomCode: "test123",
		MaxVisits:  100,
	}

	shortURL, err := us.Create(context.Background(), req)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if shortURL.Code != "test123" {
		t.Fatalf("expected code test123, got %s", shortURL.Code)
	}

	got, err := s.Get("test123")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if got.RawURL != "http://example.com/test" {
		t.Fatalf("expected raw URL http://example.com/test, got %s", got.RawURL)
	}

	redirReq := &model.RedirectRequest{
		Code:      "test123",
		Timestamp: time.Now(),
	}
	result, err := rs.HandleRedirect(context.Background(), redirReq)
	if err != nil {
		t.Fatalf("HandleRedirect failed: %v", err)
	}
	if result.RawURL != "http://example.com/test" {
		t.Fatalf("expected raw URL http://example.com/test, got %s", result.RawURL)
	}
	if result.Status != 302 {
		t.Fatalf("expected status 302, got %d", result.Status)
	}

	fmt.Println("GREEN (绿灯，缺陷已修复)")
	fmt.Println("所有测试通过，context 取消传播正常工作")
}
