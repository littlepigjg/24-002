package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"time"

	"logalert/internal/config"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

// HealthHandler handles health check and readiness probe requests.
type HealthHandler struct {
	logger    logger.Logger
	startTime time.Time
	urlStore  *store.URLStore
	cfg       *config.Config
}

// NewHealthHandler creates a new HealthHandler.
func NewHealthHandler(log logger.Logger, us *store.URLStore, cfg *config.Config) *HealthHandler {
	return &HealthHandler{
		logger:    log.WithField("handler", "health"),
		startTime: time.Now(),
		urlStore:  us,
		cfg:       cfg,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status    string      `json:"status"`
	Uptime    string      `json:"uptime"`
	Version   string      `json:"version"`
	GoVersion string      `json:"go_version"`
	Memory    MemoryStats `json:"memory"`
	Goroutines int        `json:"goroutines"`
	Timestamp time.Time   `json:"timestamp"`
}

// MemoryStats contains Go runtime memory statistics.
type MemoryStats struct {
	Alloc     uint64 `json:"alloc"`
	HeapAlloc uint64 `json:"heap_alloc"`
	HeapSys   uint64 `json:"heap_sys"`
	NumGC     uint32 `json:"num_gc"`
}

// HandleHealth handles GET /health
func (h *HealthHandler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if h.urlStore != nil {
		ctx, cancel := getHealthCheckContext(r.Context(), h.cfg)
		defer cancel()

		if err := h.urlStore.Load(ctx); err != nil {
			h.logger.Error("health check load failed", "error", err)
		}
	}

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

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// HandleReady handles GET /ready
func (h *HealthHandler) HandleReady(w http.ResponseWriter, r *http.Request) {
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
