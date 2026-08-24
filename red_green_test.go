package main

import (
	"fmt"
	"testing"
	"time"

	"logalert/pkg/timeutil"
)

func TestRedGreen(t *testing.T) {
	shanghai, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatal("failed to load timezone:", err)
	}

	newYork, _ := time.LoadLocation("America/New_York")

	logTime := time.Date(2026, 8, 24, 15, 0, 0, 0, shanghai)
	if logTime.UTC().Hour() != 7 {
		t.Skip("unexpected Shanghai timezone offset, skipping test")
		return
	}

	nyTime := time.Date(2026, 8, 24, 2, 0, 0, 0, newYork)
	if nyTime.UTC().Hour() != 6 {
		t.Skip("unexpected NY timezone offset, skipping test")
		return
	}

	laTime := time.Date(2026, 8, 23, 23, 0, 0, 0, time.FixedZone("LAX", -7*3600))
	if laTime.UTC().Hour() != 6 {
		t.Skip("unexpected LA timezone offset, skipping test")
		return
	}

	windowStart := time.Date(2026, 8, 24, 7, 0, 0, 0, time.UTC)
	windowEnd := time.Date(2026, 8, 24, 8, 0, 0, 0, time.UTC)
	windowSize := time.Hour

	allGreen := true

	w := timeutil.NewTimeWindow(windowStart, windowSize)
	if w.Contains(logTime) {
		fmt.Println("Test 1 (TimeWindow.Contains跨时区): GREEN")
	} else {
		fmt.Println("Test 1 (TimeWindow.Contains跨时区): RED")
		allGreen = false
	}

	r := timeutil.NewTimeRange(windowStart, windowEnd)
	if r.Contains(logTime) {
		fmt.Println("Test 2 (TimeRange.Contains跨时区): GREEN")
	} else {
		fmt.Println("Test 2 (TimeRange.Contains跨时区): RED")
		allGreen = false
	}

	w2 := timeutil.NewTimeWindow(logTime, time.Hour)
	if w.Overlaps(w2) {
		fmt.Println("Test 3 (TimeWindow.Overlaps跨时区): GREEN")
	} else {
		fmt.Println("Test 3 (TimeWindow.Overlaps跨时区): RED")
		allGreen = false
	}

	r2 := timeutil.NewTimeRange(logTime, logTime.Add(time.Hour))
	if r.Overlaps(r2) {
		fmt.Println("Test 4 (TimeRange.Overlaps跨时区): GREEN")
	} else {
		fmt.Println("Test 4 (TimeRange.Overlaps跨时区): RED")
		allGreen = false
	}

	if r.IsValid() {
		fmt.Println("Test 5 (TimeRange.IsValid): GREEN")
	} else {
		fmt.Println("Test 5 (TimeRange.IsValid): RED")
		allGreen = false
	}

	nyWindow := timeutil.NewTimeWindow(nyTime, time.Hour)
	if nyWindow.Contains(nyTime) {
		fmt.Println("Test 6 (Window自身一致性): GREEN")
	} else {
		fmt.Println("Test 6 (Window自身一致性): RED")
		allGreen = false
	}

	tr := timeutil.NewTimeRange(windowStart, windowEnd)
	hourlyBuckets := timeutil.HourlyBuckets(tr)

	targetHour := 7
	correctBucketContains := false
	for _, b := range hourlyBuckets {
		if b.Start.Hour() == targetHour {
			if b.Contains(logTime) {
				correctBucketContains = true
			}
			break
		}
	}
	if correctBucketContains {
		fmt.Println("Test 7 (HourlyBuckets正确桶包含): GREEN")
	} else {
		fmt.Println("Test 7 (HourlyBuckets正确桶包含): RED")
		allGreen = false
	}

	iter := timeutil.NewWindowIterator(tr, 30*time.Minute)
	found := false
	for iter.Next() {
		w := iter.Window()
		if w.Contains(logTime) {
			found = true
			break
		}
		iter.Advance()
	}
	if found {
		fmt.Println("Test 8 (WindowIterator跨时区): GREEN")
	} else {
		fmt.Println("Test 8 (WindowIterator跨时区): RED")
		allGreen = false
	}

	nyWindowStart := time.Date(2026, 8, 24, 6, 0, 0, 0, time.UTC)
	nyW := timeutil.NewTimeWindow(nyWindowStart, time.Hour)
	if nyW.Contains(nyTime) {
		fmt.Println("Test 9 (多时区New York): GREEN")
	} else {
		fmt.Println("Test 9 (多时区New York): RED")
		allGreen = false
	}

	laWindowStart := time.Date(2026, 8, 24, 6, 0, 0, 0, time.UTC)
	laW := timeutil.NewTimeWindow(laWindowStart, time.Hour)
	if laW.Contains(laTime) {
		fmt.Println("Test 10 (多时区Los Angeles): GREEN")
	} else {
		fmt.Println("Test 10 (多时区Los Angeles): RED")
		allGreen = false
	}

	if allGreen {
		fmt.Println("GREEN (绿灯，缺陷已修复)")
	} else {
		fmt.Println("RED (红灯，缺陷未修复)")
		t.Fail()
	}
}
