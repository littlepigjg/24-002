package store

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// newTestStore builds a MemoryLogStore backed by a discard logger so tests
// stay quiet and fast.
func newTestStore(maxSize int) *MemoryLogStore {
	return NewMemoryLogStore(maxSize, logger.NewLogger(logger.LogLevelError, logger.NewDiscardWriter()))
}

// makeEntry builds a log entry with a unique ID so concurrent writers never
// collide on the same key (isolating data-loss checks from key collisions).
func makeEntry(i int) *model.LogEntry {
	return &model.LogEntry{
		ID:        fmt.Sprintf("entry-%d", i),
		Timestamp: time.Unix(int64(i), 0),
		Level:     model.LevelInfo,
		Source:    "svc-a",
		Message:   "msg",
		Service:   "svc-a",
		Tags:      map[string]string{},
	}
}

// TestConcurrentStoreAndScan_NoDataLoss hammers Store from several goroutines
// while a rule-scan goroutine repeatedly calls RawSnapshot (the exact pattern
// the scheduler uses). Total writes stay below maxSize, so nothing is evicted
// and every successful write must be present in the final snapshot.
// Under the old code this raced (concurrent map read/write panic) and lost
// data (TOCTOU around eviction+write).
func TestConcurrentStoreAndScan_NoDataLoss(t *testing.T) {
	const writers = 8
	const perWriter = 400 // 3.2k writes total, well under maxSize below
	maxSize := perWriter*writers + 2000 // no eviction expected

	store := newTestStore(maxSize)

	var writerWG sync.WaitGroup
	writerWG.Add(writers)

	// Rule-scan goroutine: mirror scheduler.evaluateRule's RawSnapshot path.
	// Bounded iterations with a tiny yield so it doesn't monopolize the
	// read lock and starve writers under the race detector.
	var writes atomic.Int64
	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		for i := 0; i < 2000; i++ {
			snap := store.RawSnapshot()
			for _, e := range snap { // iterate read-only like countFromSnapshot
				_ = e.Level
				_ = e.Source
				_ = e.Timestamp
			}
			time.Sleep(50 * time.Microsecond)
		}
	}()

	for w := 0; w < writers; w++ {
		go func(w int) {
			defer writerWG.Done()
			for i := 0; i < perWriter; i++ {
				idx := w*perWriter + i
				if err := store.Store(context.Background(), makeEntry(idx)); err != nil {
					t.Errorf("Store failed: %v", err)
					return
				}
				writes.Add(1)
			}
		}(w)
	}

	writerWG.Wait()
	<-scanDone

	// No eviction should have occurred: every successful write must be present.
	snap := store.RawSnapshot()
	if got, want := len(snap), int(writes.Load()); got != want {
		t.Fatalf("data loss: snapshot has %d entries, but %d writes succeeded", got, want)
	}
}

// TestConcurrentStoreBatchAndCount exercises StoreBatch + Count concurrently,
// covering the other racy path (StoreBatch previously mutated entry.Timestamp
// unlocked between two lock acquisitions).
func TestConcurrentStoreBatchAndCount(t *testing.T) {
	const writers = 6
	const batches = 80
	const batchSize = 8
	maxSize := writers*batches*batchSize + 2000

	store := newTestStore(maxSize)

	var writerWG sync.WaitGroup
	writerWG.Add(writers + 1)

	go func() {
		defer writerWG.Done()
		filter := &model.LogFilter{}
		for i := 0; i < 1000; i++ {
			if _, err := store.Count(context.Background(), filter); err != nil {
				t.Errorf("Count failed: %v", err)
				return
			}
			time.Sleep(50 * time.Microsecond)
		}
	}()

	for w := 0; w < writers; w++ {
		go func(w int) {
			defer writerWG.Done()
			for b := 0; b < batches; b++ {
				entries := make([]*model.LogEntry, batchSize)
				for i := 0; i < batchSize; i++ {
					entries[i] = makeEntry(w*batches*batchSize + b*batchSize + i)
				}
				if err := store.StoreBatch(context.Background(), entries); err != nil {
					t.Errorf("StoreBatch failed: %v", err)
					return
				}
			}
		}(w)
	}

	writerWG.Wait()
}

// TestEvictionUnderConcurrency verifies that once capacity is exceeded,
// eviction kicks in and the store never exceeds maxSize, and that concurrent
// writers + scanners do not race during eviction.
func TestEvictionUnderConcurrency(t *testing.T) {
	maxSize := 500
	store := newTestStore(maxSize)

	var writerWG sync.WaitGroup
	writerWG.Add(maxSize) // fixed, modest goroutine count

	scanDone := make(chan struct{})
	go func() {
		defer close(scanDone)
		for i := 0; i < 1000; i++ {
			snap := store.RawSnapshot()
			for _, e := range snap {
				_ = e.Level
			}
			time.Sleep(50 * time.Microsecond)
		}
	}()

	for i := 0; i < maxSize; i++ {
		go func(i int) {
			defer writerWG.Done()
			_ = store.Store(context.Background(), makeEntry(i))
		}(i)
	}

	writerWG.Wait()
	<-scanDone

	snap := store.RawSnapshot()
	if len(snap) > maxSize {
		t.Fatalf("store exceeded capacity: %d > %d", len(snap), maxSize)
	}
	if len(snap) == 0 {
		t.Fatal("store is empty after eviction; expected some surviving entries")
	}
}

// TestEvictNowIsRaceFree drives EvictNow concurrently with writers and
// scanners. EvictNow previously iterated/mutated entries unlocked.
func TestEvictNowIsRaceFree(t *testing.T) {
	maxSize := 500
	store := newTestStore(maxSize)

	// Seed with entries so EvictNow has work.
	for i := 0; i < maxSize; i++ {
		_ = store.Store(context.Background(), makeEntry(i))
	}

	var wg sync.WaitGroup
	wg.Add(3)

	go func() {
		defer wg.Done()
		for i := 0; i < maxSize; i++ {
			_ = store.Store(context.Background(), makeEntry(maxSize+i))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			_, _ = store.EvictNow(context.Background())
			time.Sleep(50 * time.Microsecond)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 500; i++ {
			snap := store.RawSnapshot()
			for _, e := range snap {
				_ = e.Level
			}
			time.Sleep(50 * time.Microsecond)
		}
	}()

	wg.Wait()
}
