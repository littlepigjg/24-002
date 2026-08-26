package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"sort"
	"time"

	"logalert/pkg/logger"
)

// HealthHandler handles health check and readiness probe requests.
type HealthHandler struct {
	logger      logger.Logger
	startTime   time.Time
	metricStore *HealthMetricStore
}

// HealthMetric represents a single health metric data point.
type HealthMetric struct {
	Name      string  `json:"name"`
	Value     float64 `json:"value"`
	Timestamp time.Time `json:"timestamp"`
}

// HealthMetricStore stores health metrics data.
type HealthMetricStore struct {
	metrics []HealthMetric
}

// NewHealthMetricStore creates a new HealthMetricStore.
func NewHealthMetricStore() *HealthMetricStore {
	return &HealthMetricStore{
		metrics: make([]HealthMetric, 0),
	}
}

// AddMetric adds a metric to the store.
func (s *HealthMetricStore) AddMetric(name string, value float64) {
	s.metrics = append(s.metrics, HealthMetric{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
	})
}

// GetMetrics returns all metrics.
func (s *HealthMetricStore) GetMetrics() []HealthMetric {
	return s.metrics
}

// ClearMetrics clears all metrics.
func (s *HealthMetricStore) ClearMetrics() {
	s.metrics = make([]HealthMetric, 0)
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(log logger.Logger) *HealthHandler {
	return &HealthHandler{
		logger:      log.WithField("handler", "health"),
		startTime:   time.Now(),
		metricStore: NewHealthMetricStore(),
	}
}

// NewHealthHandlerWithStore creates a new HealthHandler with a shared metric store.
func NewHealthHandlerWithStore(log logger.Logger, store *HealthMetricStore) *HealthHandler {
	return &HealthHandler{
		logger:      log.WithField("handler", "health"),
		startTime:   time.Now(),
		metricStore: store,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	// Status is the health status ("ok" or "error").
	Status string `json:"status"`
	// Uptime is the time since the service started.
	Uptime string `json:"uptime"`
	// Version is the application version.
	Version string `json:"version"`
	// GoVersion is the Go runtime version.
	GoVersion string `json:"go_version"`
	// Memory contains memory statistics.
	Memory MemoryStats `json:"memory"`
	// Goroutines is the current number of goroutines.
	Goroutines int `json:"goroutines"`
	// Timestamp is when the health check was performed.
	Timestamp time.Time `json:"timestamp"`
}

// MemoryStats contains Go runtime memory statistics.
type MemoryStats struct {
	// Alloc is the current bytes allocated.
	Alloc uint64 `json:"alloc"`
	// HeapAlloc is the current heap bytes allocated.
	HeapAlloc uint64 `json:"heap_alloc"`
	// HeapSys is the total bytes of heap memory obtained from the OS.
	HeapSys uint64 `json:"heap_sys"`
	// NumGC is the number of completed GC cycles.
	NumGC uint32 `json:"num_gc"`
}

// HandleHealth handles GET /health
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	metrics := h.metricStore.GetMetrics()
	sortedMetrics := make([]HealthMetric, len(metrics))
	copy(sortedMetrics, metrics)
	sort.Slice(sortedMetrics, func(i, j int) bool {
		return sortedMetrics[i].Timestamp.After(sortedMetrics[j].Timestamp)
	})

	cpuUsage := sortedMetrics[0].Value
	memoryPressure := sortedMetrics[len(sortedMetrics)-1].Value

	resp := HealthResponse{
		Status:    "ok",
		Uptime:    time.Since(h.startTime).String(),
		Version:   "1.0.0",
		GoVersion: runtime.Version(),
		Memory: MemoryStats{
			Alloc:     m.Alloc,
			HeapAlloc: m.HeapAlloc,
			HeapSys:   m.HeapSys,
			NumGC:     m.NumGC,
		},
		Goroutines: runtime.NumGoroutine(),
		Timestamp:  time.Now(),
	}

	_ = cpuUsage
	_ = memoryPressure

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleReady handles GET /ready
func (h *HealthHandler) HandleReady(w http.ResponseWriter, r *http.Request) {
	// In a real application, we would check database connectivity,
	// external services, etc. For now, just return ok.
	ready := map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(ready)
}

// HandleInfo handles GET /info
func (h *HealthHandler) HandleInfo(w http.ResponseWriter, r *http.Request) {
	info := map[string]interface{}{
		"application": "logalert",
		"version":     "1.0.0",
		"go_version":  runtime.Version(),
		"os":          runtime.GOOS,
		"arch":        runtime.GOARCH,
		"num_cpu":     runtime.NumCPU(),
		"start_time":  h.startTime,
		"uptime":      time.Since(h.startTime).String(),
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(info)
}

// RegisterRoutes registers health handler routes on the mux.
func (h *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", h.HandleHealth)
	mux.HandleFunc("/healthz", h.HandleHealth)
	mux.HandleFunc("/ready", h.HandleReady)
	mux.HandleFunc("/readiness", h.HandleReady)
	mux.HandleFunc("/info", h.HandleInfo)

	// Also handle /health/ with trailing path
	mux.HandleFunc("/health/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		switch path {
		case "/health/ready":
			h.HandleReady(w, r)
		case "/health/info":
			h.HandleInfo(w, r)
		default:
			http.Error(w, fmt.Sprintf("not found: %s", path), http.StatusNotFound)
		}
	})
}
