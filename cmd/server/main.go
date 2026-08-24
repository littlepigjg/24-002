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
	logWriter := logger.NewStdoutWriter()
	log := logger.NewLogger(logger.LogLevelInfo, logWriter)

	log.Info("starting logalert server")

	configPath := os.Getenv("CONFIG_PATH")
	cfg, err := config.Load(configPath)
	if err != nil {
		log.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	logLevel := logger.ParseLogLevel(cfg.Logging.Level)
	log.SetLevel(logLevel)

	log.Info("configuration loaded", "port", cfg.Server.Port, "log_level", cfg.Logging.Level)

	logStore := store.NewMemoryLogStore(cfg.Storage.MaxLogEntries, log)
	ruleStore := store.NewMemoryRuleStore(log)
	alertStore := store.NewMemoryAlertStore(cfg.Storage.MaxAlertRecords, log)

	logSvc := service.NewLogService(logStore, cfg, log)
	ruleSvc := service.NewRuleService(ruleStore, cfg, log)
	alertSvc := service.NewAlertService(alertStore, cfg, log)
	statsSvc := service.NewStatsService(logStore, cfg, log)

	scheduler := service.NewScheduler(ruleSvc, alertSvc, logStore, cfg, log)

	mw := handler.NewMiddleware(log)
	logHandler := handler.NewLogHandler(logSvc, log)
	ruleHandler := handler.NewRuleHandler(ruleSvc, log)
	alertHandler := handler.NewAlertHandler(alertSvc, log)
	statsHandler := handler.NewStatsHandler(statsSvc, log)
	healthHandler := handler.NewHealthHandler(log)
	schedulerHandler := handler.NewSchedulerHandler(scheduler, log)

	mux := http.NewServeMux()

	logHandler.RegisterRoutes(mux)
	ruleHandler.RegisterRoutes(mux)
	alertHandler.RegisterRoutes(mux)
	statsHandler.RegisterRoutes(mux)
	healthHandler.RegisterRoutes(mux)
	schedulerHandler.RegisterRoutes(mux)

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" || r.URL.Path == "/index.html" {
			http.ServeFile(w, r, "static/index.html")
			return
		}
		http.NotFound(w, r)
	})

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

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      handlerChain,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

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

	if cfg.Scheduler.EnableAutoScan {
		if err := scheduler.Start(ctx, &wg); err != nil {
			log.Error("failed to start scheduler", "error", err)
		}
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("received signal, shutting down", "signal", sig)
	case <-ctx.Done():
		log.Info("context cancelled, shutting down")
	}

	log.Info("shutting down gracefully")

	scheduler.Stop()

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error("server shutdown failed", "error", err)
	}

	logStore.Close()
	ruleStore.Close()
	alertStore.Close()

	wg.Wait()

	log.Info("server shutdown complete")
}

func jsonResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(data)
}

func Now() time.Time {
	return time.Now()
}
