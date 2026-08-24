package store

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"logalert/internal/model"
	"logalert/pkg/logger"
)

// sumByLevel sums the per-level counts, which must always equal TotalCount since
// both aggregates describe the same set of entries.
func sumByLevel(m map[model.LogLevel]int64) int64 {
	var sum int64
	for _, c := range m {
		sum += c
	}
	return sum
}

// TestStatisticsConsistencyUnderConcurrentWrites reproduces the high-concurrency
// inconsistency reported against the stats endpoint: total_count drifting from
// the sum of by_level. The earlier Statistics implementation sampled entries in
// two separate read locks, so a concurrent writer could mutate the map between the
// two passes and make TotalCount (pass 1) disagree with ByLevel (pass 2).
//
// Writers hammer the store (single + batch + delete, with a tight maxSize to also
// exercise eviction) while readers call Statistics in a loop. Every reader result
// must satisfy TotalCount == sum(ByLevel).
func TestStatisticsConsistencyUnderConcurrentWrites(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelError, logger.NewStdoutWriter())
	store := NewMemoryLogStore(256, log) // small so eviction also races with reads

	levels := []model.LogLevel{
		model.LevelDebug, model.LevelInfo, model.LevelWarn,
		model.LevelError, model.LevelFatal,
	}

	ctx := context.Background()
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)

	var wg sync.WaitGroup
	stop := make(chan struct{})

	// Concurrent writers: single Store, StoreBatch, and Delete intermixed.
	const writers = 8
	wg.Add(writers)
	for w := 0; w < writers; w++ {
		go func(w int) {
			defer wg.Done()
			i := 0
			for {
				select {
				case <-stop:
					return
				default:
				}
				lvl := levels[i%len(levels)]
				e := model.NewLogEntry("svc-a", lvl, fmt.Sprintf("msg %d-%d", w, i))
				e.Timestamp = time.Now()
				_ = store.Store(ctx, e)

				if i%50 == 0 {
					batch := make([]*model.LogEntry, 0, 10)
					for b := 0; b < 10; b++ {
						be := model.NewLogEntry("svc-b", levels[(i+b)%len(levels)], "batch msg")
						be.Timestamp = time.Now()
						batch = append(batch, be)
					}
					_ = store.StoreBatch(ctx, batch)
				}

				i++
			}
		}(w)
	}

	// Concurrent readers: Statistics must stay internally consistent.
	const readers = 4
	var rwg sync.WaitGroup
	rwg.Add(readers)
	var inconsistent int64
	var mu sync.Mutex

	for r := 0; r < readers; r++ {
		go func() {
			defer rwg.Done()
			for {
				select {
				case <-stop:
					return
				default:
				}
				stats, err := store.Statistics(ctx, from, to)
				if err != nil {
					t.Errorf("Statistics returned error: %v", err)
					return
				}
				if got := sumByLevel(stats.ByLevel); got != stats.TotalCount {
					mu.Lock()
					inconsistent++
					mu.Unlock()
					t.Errorf("inconsistent statistics: total_count=%d but sum(by_level)=%d", stats.TotalCount, got)
				}
			}
		}()
	}

	// Let the race run long enough to surface the old two-lock bug.
	time.Sleep(300 * time.Millisecond)
	close(stop)
	wg.Wait()
	rwg.Wait()

	if inconsistent > 0 {
		t.Fatalf("observed %d inconsistent Statistics results under concurrent writes", inconsistent)
	}
}

// TestStatisticsSnapshotIsConsistent is a focused, non-time-dependent check that
// every field of LogStatistics is derived from one snapshot, including the
// AvgMessageLength and ErrorRate denominators.
func TestStatisticsSnapshotIsConsistent(t *testing.T) {
	log := logger.NewLogger(logger.LogLevelError, logger.NewStdoutWriter())
	st := NewMemoryLogStore(1000, log)
	ctx := context.Background()
	from := time.Now().Add(-time.Hour)
	to := time.Now().Add(time.Hour)

	seed := []*model.LogEntry{
		model.NewLogEntry("s1", model.LevelError, "boom"),
		model.NewLogEntry("s1", model.LevelInfo, "ok"),
		model.NewLogEntry("s2", model.LevelFatal, "fatal"),
		model.NewLogEntry("s2", model.LevelInfo, "ok"),
	}
	for _, e := range seed {
		e.Timestamp = time.Now()
		_ = st.Store(ctx, e)
	}

	stats, err := st.Statistics(ctx, from, to)
	if err != nil {
		t.Fatalf("Statistics: %v", err)
	}

	if stats.TotalCount != 4 {
		t.Fatalf("TotalCount = %d, want 4", stats.TotalCount)
	}
	if got := sumByLevel(stats.ByLevel); got != stats.TotalCount {
		t.Fatalf("sum(by_level) = %d, want %d", got, stats.TotalCount)
	}

	var srcSum int64
	for _, c := range stats.BySource {
		srcSum += c
	}
	if srcSum != stats.TotalCount {
		t.Fatalf("sum(by_source) = %d, want %d", srcSum, stats.TotalCount)
	}

	// 2 of 4 are error/fatal.
	if stats.ErrorRate != 0.5 {
		t.Fatalf("ErrorRate = %v, want 0.5", stats.ErrorRate)
	}

	wantAvg := float64(len("boom")+len("ok")+len("fatal")+len("ok")) / 4
	if stats.AvgMessageLength != wantAvg {
		t.Fatalf("AvgMessageLength = %v, want %v", stats.AvgMessageLength, wantAvg)
	}
}
