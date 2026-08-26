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
	log := logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter())
	s := store.NewMemoryLogStore(200, log)
	ctx := context.Background()

	for i := 0; i < 120; i++ {
		entry := model.NewLogEntry(
			fmt.Sprintf("svc-%d", i%5),
			model.LevelInfo,
			fmt.Sprintf("log message number %d", i),
		)
		entry.Timestamp = time.Now().Add(-time.Duration(120-i) * time.Second)
		if err := s.Store(ctx, entry); err != nil {
			t.Fatalf("failed to store entry %d: %v", i, err)
		}
	}

	filter := &model.LogFilter{}

	page1, err := s.Query(ctx, filter, 10, 0)
	if err != nil {
		t.Fatalf("query page1 failed: %v", err)
	}
	if len(page1) != 10 {
		t.Fatalf("expected 10 results for page1, got %d", len(page1))
	}

	page1IDs := make(map[string]bool)
	for _, e := range page1 {
		if e != nil {
			page1IDs[e.ID] = true
		}
	}

	page1 = append(page1, nil)

	page2, err := s.Query(ctx, filter, 10, 10)
	if err != nil {
		t.Fatalf("query page2 failed: %v", err)
	}
	if len(page2) != 10 {
		t.Fatalf("expected 10 results for page2, got %d", len(page2))
	}

	hasNil := false
	hasOverlap := false
	for _, e := range page2 {
		if e == nil {
			hasNil = true
			continue
		}
		if page1IDs[e.ID] {
			hasOverlap = true
		}
	}

	if hasNil || hasOverlap {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Errorf("分页数据异常 - hasNil=%v hasOverlap=%v", hasNil, hasOverlap)
	} else {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	}
}

func TestRedGreenMultiplePages(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter())
	s := store.NewMemoryLogStore(200, log)
	ctx := context.Background()

	for i := 0; i < 150; i++ {
		entry := model.NewLogEntry(
			fmt.Sprintf("svc-%d", i%3),
			model.LevelInfo,
			fmt.Sprintf("entry-%d", i),
		)
		entry.Timestamp = time.Now().Add(-time.Duration(150-i) * time.Second)
		if err := s.Store(ctx, entry); err != nil {
			t.Fatalf("failed to store entry %d: %v", i, err)
		}
	}

	filter := &model.LogFilter{}

	allIDs := make(map[string]int)

	for page := 0; page < 15; page++ {
		results, err := s.Query(ctx, filter, 10, page*10)
		if err != nil {
			t.Fatalf("query page %d failed: %v", page, err)
		}
		if len(results) == 0 {
			break
		}

		for _, e := range results {
			if e == nil {
				fmt.Printf("RED (红灯，缺陷未修复)\n")
				t.Errorf("page %d contains nil entry", page)
				return
			}
			if _, exists := allIDs[e.ID]; exists {
				fmt.Printf("RED (红灯，缺陷未修复)\n")
				t.Errorf("duplicate entry ID %s found across pages", e.ID)
				return
			}
			allIDs[e.ID] = page
		}

		results = append(results, nil)
	}

	if len(allIDs) == 150 {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Printf("RED (红灯，缺陷未修复)\n")
		t.Errorf("expected 150 unique entries, got %d", len(allIDs))
	}
}