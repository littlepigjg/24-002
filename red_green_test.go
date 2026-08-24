package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"logalert/internal/config"
	"logalert/internal/model"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Storage.DataDir = t.TempDir()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	done := make(chan struct{})
	go func() {
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			us, _ := store.NewURLStoreWithCacheSize(cfg, 1)
			ls, _ := store.NewAccessLogStore(cfg)
			ls.Open(context.Background())

			urlSvc, _ := service.NewURLService(cfg, us)

			for i := 0; i < 1; i++ {
				req := &model.CreateReq{RawURL: "https://example.com"}
				urlSvc.Create(context.Background(), req)
			}

			for i := 0; i < 3; i++ {
				req := &model.CreateReq{
					RawURL: fmt.Sprintf("https://test%d.com", i),
				}
				urlSvc.Create(ctx, req)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			fp, _ := store.NewFilePersistence(t.TempDir(), logger.Default())

			ctx2, cancel2 := context.WithCancel(context.Background())
			cancel2()

			go func() {
				fp.SaveLogs(ctx2, nil)
			}()

			time.Sleep(50 * time.Millisecond)
			fp.SaveLogs(context.Background(), nil)
		}()

		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	case <-time.After(3 * time.Second):
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Fatalf("deadlock detected: operations did not complete within timeout")
	}
}
