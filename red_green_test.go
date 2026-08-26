package main

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func TestRedGreen(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter())
	s := store.NewMemoryLogStore(100000, log)

	baseTime := time.Now().Add(-1 * time.Hour)
	for i := 0; i < 200; i++ {
		entry := model.NewLogEntry("svc-a", model.LevelInfo, fmt.Sprintf("msg-%d", i))
		entry.Timestamp = baseTime.Add(time.Duration(i) * time.Second)
		entry.Service = "svc-a"
		if i%3 == 0 {
			entry.Level = model.LevelError
		} else if i%3 == 1 {
			entry.Level = model.LevelWarn
		}
		s.Store(context.Background(), entry)
	}

	var wg sync.WaitGroup
	stop := make(chan struct{})

	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-stop:
				return
			default:
				entry := model.NewLogEntry("writer", model.LevelError, "concurrent-write")
				entry.Timestamp = time.Now()
				entry.Service = "writer"
				s.Store(context.Background(), entry)
			}
		}
	}()

	time.Sleep(50 * time.Millisecond)

	foundInconsistency := false
	now := time.Now()
	for i := 0; i < 100; i++ {
		stats, err := s.Statistics(context.Background(), now.Add(-2*time.Hour), now.Add(2*time.Second))
		if err != nil {
			t.Errorf("Statistics failed: %v", err)
			foundInconsistency = true
			break
		}

		var levelSum int64
		for _, count := range stats.ByLevel {
			levelSum += count
		}

		if stats.TotalCount != levelSum {
			foundInconsistency = true
			break
		}
	}

	close(stop)
	wg.Wait()

	if foundInconsistency {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.FailNow()
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}
