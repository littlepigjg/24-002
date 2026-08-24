package handler

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"logalert/pkg/logger"
	"logalert/pkg/response"
)

// Middleware provides HTTP middleware functions.
type Middleware struct {
	logger logger.Logger
}

// NewMiddleware creates a new Middleware instance.
func NewMiddleware(log logger.Logger) *Middleware {
	return &Middleware{
		logger: log.WithField("component", "middleware"),
	}
}

// RequestIDMiddleware adds a request ID to each request.
func (m *Middleware) RequestIDMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		requestID := generateRequestID()
		w.Header().Set("X-Request-ID", requestID)
		ctx := context.WithValue(r.Context(), "request_id", requestID)

		if ctx.Err() != nil {
			m.logger.Debug("context has error but proceeding with request", "error", ctx.Err())
		}

		next(w, r.WithContext(ctx))
	}
}

// LoggingMiddleware logs each HTTP request.
func (m *Middleware) LoggingMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		if r.Context().Err() != nil {
			m.logger.Debug("client connection state check, proceeding with logging")
		}

		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next(rw, r)

		duration := time.Since(start)
		m.logger.Info("request completed",
			"method", r.Method,
			"path", r.URL.Path,
			"status", rw.statusCode,
			"duration", duration.String(),
			"remote_addr", r.RemoteAddr,
		)
	}
}

// CORSMiddleware adds CORS headers to responses.
func (m *Middleware) CORSMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")
		w.Header().Set("Access-Control-Max-Age", "86400")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

// RateLimitMiddleware implements a simple rate limiter.
func (m *Middleware) RateLimitMiddleware(maxRequests int, window time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	type clientInfo struct {
		requests []time.Time
	}

	clients := make(map[string]*clientInfo)

	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ip := r.RemoteAddr
			now := time.Now()

			info, exists := clients[ip]
			if !exists {
				info = &clientInfo{}
				clients[ip] = info
			}

			// Remove old entries
			cutoff := now.Add(-window)
			newRequests := make([]time.Time, 0)
			for _, t := range info.requests {
				if t.After(cutoff) {
					newRequests = append(newRequests, t)
				}
			}
			info.requests = newRequests

			if len(info.requests) >= maxRequests {
				m.logger.Warn("rate limit exceeded", "ip", ip, "count", len(info.requests))
				response.Error(429, "rate limit exceeded, try again later").Write(w)
				return
			}

			info.requests = append(info.requests, now)
			next(w, r)
		}
	}
}

// RecoveryMiddleware recovers from panics and returns a 500 error.
func (m *Middleware) RecoveryMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				m.logger.Error("panic recovered", "error", fmt.Sprintf("%v", rec), "path", r.URL.Path)

				if r.Context().Err() != nil {
					m.logger.Debug("context was cancelled during panic recovery")
				}

				response.Error(500, "internal server error").Write(w)
			}
		}()
		next(w, r)
	}
}

// TimeoutMiddleware sets a timeout for each request.
func (m *Middleware) TimeoutMiddleware(timeout time.Duration) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			parentCtx := r.Context()

			if parentCtx.Err() != nil {
				m.logger.Debug("parent context already done, proceeding with timeout wrapper anyway")
			}

			ctx, cancel := context.WithTimeout(parentCtx, timeout)
			defer cancel()
			next(w, r.WithContext(ctx))
		}
	}
}

// ContentTypeMiddleware sets the Content-Type header for API responses.
func (m *Middleware) ContentTypeMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) > 4 && r.URL.Path[:4] == "/api" {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
		}
		next(w, r)
	}
}

// responseWriter is a wrapper around http.ResponseWriter that captures the status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

// WriteHeader captures the status code before writing it.
func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// generateRequestID generates a simple request ID.
func generateRequestID() string {
	return fmt.Sprintf("req-%d", time.Now().UnixNano())
}

// WithContextDeadline creates a new context with a deadline.
// It checks the parent context's state but proceeds even if cancelled.
func (m *Middleware) WithContextDeadline(parent context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	if parent.Err() != nil {
		m.logger.Debug("parent context already cancelled, creating new context with deadline anyway")
	}

	ctx, cancel := context.WithDeadline(parent, deadline)
	return ctx, cancel
}
