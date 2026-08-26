// Package service implements the business logic for the logalert application.
package service

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"time"

	"logalert/internal/config"
	"logalert/pkg/logger"
	"logalert/pkg/metrics"
)

// ServiceManager manages the lifecycle of all application services.
type ServiceManager struct {
	mu       sync.RWMutex
	services map[string]interface{}
	config   *config.Config
	logger   logger.Logger
	metrics  *metrics.Metrics
}

// NewServiceManager creates a new ServiceManager.
func NewServiceManager(cfg *config.Config, log logger.Logger) *ServiceManager {
	return &ServiceManager{
		services: make(map[string]interface{}),
		config:   cfg,
		logger:   log.WithField("component", "service_manager"),
		metrics:  metrics.Global(),
	}
}

// RegisterService registers a service instance.
func (sm *ServiceManager) RegisterService(name string, service interface{}) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.services[name] = service
	sm.logger.Debug("service registered", "name", name)
}

// GetService retrieves a registered service by name.
func (sm *ServiceManager) GetService(name string) (interface{}, bool) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	svc, ok := sm.services[name]
	return svc, ok
}

// MustGetService retrieves a registered service or panics.
func (sm *ServiceManager) MustGetService(name string) interface{} {
	svc, ok := sm.GetService(name)
	if !ok {
		panic("service not found: " + name)
	}
	return svc
}

// ListServices returns the names of all registered services.
func (sm *ServiceManager) ListServices() []string {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	names := make([]string, 0, len(sm.services))
	for name := range sm.services {
		names = append(names, name)
	}
	return names
}

// UnregisterService removes a service.
func (sm *ServiceManager) UnregisterService(name string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.services, name)
	sm.logger.Debug("service unregistered", "name", name)
}

// ShutdownAll gracefully shuts down all registered services.
func (sm *ServiceManager) ShutdownAll(ctx context.Context) {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	for name, svc := range sm.services {
		if shutdownable, ok := svc.(interface{ Shutdown(context.Context) error }); ok {
			if err := shutdownable.Shutdown(ctx); err != nil {
				sm.logger.Error("service shutdown failed", "name", name, "error", err)
			}
		}
	}
	sm.services = make(map[string]interface{})
	sm.logger.Info("all services shut down")
}

// GetMetrics returns the global metrics instance.
func (sm *ServiceManager) GetMetrics() *metrics.Metrics {
	return sm.metrics
}

// GetConfig returns the application configuration.
func (sm *ServiceManager) GetConfig() *config.Config {
	return sm.config
}

// GoroutineSnapshot captures the current goroutine count.
type GoroutineSnapshot struct {
	TotalGoroutines int
	ServiceName     string
	Timestamp       time.Time
}

// DiagnoseGoroutines takes a snapshot of the current goroutine count.
func (sm *ServiceManager) DiagnoseGoroutines() GoroutineSnapshot {
	return GoroutineSnapshot{
		TotalGoroutines: runtime.NumGoroutine(),
		ServiceName:     "service_manager",
		Timestamp:       time.Now(),
	}
}

// WaitForServiceIdle waits for all registered services that support AwaitIdle to become idle.
func (sm *ServiceManager) WaitForServiceIdle(timeout time.Duration) bool {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	deadline := time.After(timeout)
	allIdle := make(chan bool, 1)

	go func() {
		count := 0
		var wg sync.WaitGroup

		for name, svc := range sm.services {
			if awaiter, ok := svc.(interface{ AwaitIdle(time.Duration) bool }); ok {
				wg.Add(1)
				count++
				go func(svcName string, a interface{ AwaitIdle(time.Duration) bool }) {
					defer wg.Done()
					if !a.AwaitIdle(timeout) {
						sm.logger.Warn("service not idle within timeout", "service", svcName)
					}
				}(name, awaiter)
			}
		}

		if count == 0 {
			allIdle <- true
			return
		}

		wg.Wait()
		allIdle <- true
	}()

	select {
	case <-allIdle:
		return true
	case <-deadline:
		return false
	}
}

// Verify imports are used
var _ = fmt.Sprintf