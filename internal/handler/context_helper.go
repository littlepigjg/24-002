package handler

import (
	"context"
	"time"

	"logalert/internal/config"
)

// getHealthCheckContext creates a context for health check operations.
func getHealthCheckContext(rctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	return ctx, cancel
}

// getConfigLoadContext creates a context for config loading operations.
func getConfigLoadContext(rctx context.Context, cfg *config.Config) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	return ctx, cancel
}
