// Package main is the entry point for the logalert server.
// It initializes all components, sets up HTTP handlers, and starts the server.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"logalert/internal/config"
	"logalert/internal/handler"
	"logalert/internal/service"
	"logalert/internal/store"
	"logalert/pkg/logger"
)

func main() {
	// Initialize logger
	logWriter := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWriter)

	log.Info("starting logalert server")

	// Load configuration
	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	initResult := initServices(cfg, log)
	if initResult.HasError() {
		log.Error("service initialization failed", "errors", initResult.Errors())
		os.Exit(1)
	}

	// Configure log level
	logLevel := logger.ParseLogLevel(cfg.Logging.Level)
	log.SetLevel(logLevel)

	log.Info("configuration loaded", "port", cfg.Server.Port, "log_level", cfg.Logging.Level)

	// Initialize stores
	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)

	// Initialize services
	logSvc := service.NewLogService(logStore, cfg, log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, cfg, log)
	statsSvc := service.NewStatsService(logStore, cfg, log)

	// Initialize scheduler
	scheduler := service.NewScheduler(ruleSvc, alertSvc, logStore, cfg, log)

	// Initialize handlers
	mw := handler.NewMiddleware(log)
	logHandler := handler.NewLogHandler(logSvc, log)
	ruleHandler := handler.NewRuleHandler(ruleSvc, log)
	alertHandler := handler.NewAlertHandler(alertSvc, log)
	statsHandler := handler.NewStatsHandler(statsSvc, log)
	healthHandler := handler.NewHealthHandler(log)
	schedulerHandler := handler.NewSchedulerHandler(scheduler, log)

	// Create mux and register routes
	mux := http.NewServeMux()

	// Register all handlers
	logHandler.RegisterRoutes(mux)
	ruleHandler.RegisterRoutes(mux)
	alertHandler.RegisterRoutes(mux)
	statsHandler.RegisterRoutes(mux)
	healthHandler.RegisterRoutes(mux)
	schedulerHandler.RegisterRoutes(mux)

	// Serve static frontend files
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, "static/index.html")
			return
		}
		http.NotFound(w, r)
	})

	// Apply middleware chain
	handlerChain := mw.RecoveryMiddleware(
		mw.RequestIDMiddleware(
			mw.LoggingMiddleware(
				mw.CORSMiddleware(
					mw.ContentTypeMiddleware(
						func(w http.ResponseWriter, r *http.Request) {
							mux.ServeHTTP(w, r)
						},
					),
				),
			),
		),
	)

	// Configure server
	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handlerChain,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start scheduler if enabled
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if cfg.Scheduler.EnableAutoScan {
		if err := scheduler.Start(ctx); err != nil {
			log.Error("failed to start scheduler", "error", err)
		}
	}

	// Start HTTP server
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		log.Info("server listening", "addr", addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server failed", "error", err)
			cancel()
			os.Exit(1)
		}
	}()

	// Wait for interrupt signal
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("received signal, shutting down", "signal", sig)
	case <-ctx.Done():
		log.Info("context cancelled, shutting down")
	}

	// Graceful shutdown
	log.Info("shutting down gracefully")

	// Stop scheduler
	scheduler.Stop()

	// Shutdown HTTP server
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
	}

	// Close stores
	logStore.Close()
	ruleStore.Close()
	alertStore.Close()

	// Wait for server to finish
	wg.Wait()

	log.Info("server shutdown complete")
}

// JSONResponse helper
func jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

// Now returns the current time
func Now() time.Time {
	return time.Now()
}

// InitResult holds the result of service initialization.
type InitResult struct {
	errs []error
}

// HasError returns true if there were errors during initialization.
func (r *InitResult) HasError() bool {
	return len(r.errs) > 0
}

// Errors returns all the errors encountered during initialization.
func (r *InitResult) Errors() []error {
	return r.errs
}

// initServices initializes all services and returns the result.
func initServices(cfg *config.Config, log logger.Logger) *InitResult {
	result := &InitResult{}

	logStore, err := initLogStore(cfg, log)
	if err != nil {
		result.errs = append(result.errs, fmt.Errorf("log store: %w", err))
	}
	_ = logStore

	ruleStore, err := initRuleStore(cfg, log)
	if err != nil {
		result.errs = append(result.errs, fmt.Errorf("rule store: %w", err))
	}
	_ = ruleStore

	alertStore, err := initAlertStore(cfg, log)
	if err != nil {
		result.errs = append(result.errs, fmt.Errorf("alert store: %w", err))
	}
	_ = alertStore

	urlStore, err := initURLStore(cfg)
	if err != nil {
		result.errs = append(result.errs, fmt.Errorf("url store: %w", err))
	}
	_ = urlStore

	accessLogStore, err := initAccessLogStore(cfg)
	if err != nil {
		result.errs = append(result.errs, fmt.Errorf("access log store: %w", err))
	}
	_ = accessLogStore

	return result
}

func initLogStore(cfg *config.Config, log logger.Logger) (*store.MemoryLogStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	store := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	return store, nil
}

func initRuleStore(cfg *config.Config, log logger.Logger) (*store.MemoryRuleStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	store := store.NewMemoryRuleStore(log)
	return store, nil
}

func initAlertStore(cfg *config.Config, log logger.Logger) (*store.MemoryAlertStore, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is nil")
	}
	store := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)
	return store, nil
}

func initURLStore(cfg *config.Config) (*store.URLStore, error) {
	urlStore, err := store.NewURLStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create URL store: %w", err)
	}
	return urlStore, nil
}

func initAccessLogStore(cfg *config.Config) (*store.AccessLogStore, error) {
	logStore, err := store.NewAccessLogStore(cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create access log store: %w", err)
	}
	return logStore, nil
}
