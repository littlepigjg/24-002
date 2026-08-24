package redgreen

import (
	"fmt"
	"testing"

	"logalert/pkg/metrics"
)

func TestRedGreen(t *testing.T) {
	m := metrics.NewMetrics()

	for i := 0; i < 200; i++ {
		m.RecordRequest(i%2 == 0)
	}

	cache := m.RequestCache()
	size := cache.Len()

	if size > 100 {
		fmt.Println("RED（红灯，缺陷未修复）")
		t.Errorf("cache size %d exceeds maxSize 100: memory leak detected", size)
	} else {
		fmt.Println("GREEN（绿灯，缺陷已修复）")
	}
}