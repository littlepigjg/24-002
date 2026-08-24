package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"logalert/pkg/logger"
)

// TestTimeoutMiddlewareShortCircuitsCancelledParent reproduces the reported
// issue: when the parent (request) context is already cancelled — the client
// disconnected — TimeoutMiddleware used to keep deriving a child context and
// running the handler. It must now skip the handler entirely.
func TestTimeoutMiddlewareShortCircuitsCancelledParent(t *testing.T) {
	mw := NewMiddleware(logger.NewLogger(logger.LogLevelWarn, logger.NewStdoutWriter()))

	handlerCalled := false
	wrapped := mw.TimeoutMiddleware(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	// Cancel the request context before the middleware sees it.
	cancelCtx, cancel := context.WithCancel(req.Context())
	cancel()
	req = req.WithContext(cancelCtx)

	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if handlerCalled {
		t.Fatalf("handler ran for a cancelled parent context; want it short-circuited")
	}
}

// TestTimeoutMiddlewareRunsForLiveContext ensures the short-circuit does not
// break the normal path: a live parent context still drives the handler.
func TestTimeoutMiddlewareRunsForLiveContext(t *testing.T) {
	mw := NewMiddleware(logger.NewLogger(logger.LogLevelWarn, logger.NewStdoutWriter()))

	handlerCalled := false
	wrapped := mw.TimeoutMiddleware(time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		handlerCalled = true
		// The derived context must be live while the handler runs.
		if err := r.Context().Err(); err != nil {
			t.Errorf("derived context already done: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/logs", nil)
	rec := httptest.NewRecorder()
	wrapped.ServeHTTP(rec, req)

	if !handlerCalled {
		t.Fatalf("handler did not run for a live context")
	}
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d; want 200", rec.Code)
	}
}

// TestWithContextDeadlineRefusesCancelledParent verifies that
// WithContextDeadline does not paper over an already-done parent by creating a
// fresh deadline context; it returns a context that is immediately done.
func TestWithContextDeadlineRefusesCancelledParent(t *testing.T) {
	mw := NewMiddleware(logger.NewLogger(logger.LogLevelWarn, logger.NewStdoutWriter()))

	parent, cancel := context.WithCancel(context.Background())
	cancel()

	ctx, cleanup := mw.WithContextDeadline(parent, time.Now().Add(time.Hour))
	defer cleanup()
	if ctx.Err() == nil {
		t.Fatalf("expected derived context to be immediately done for a cancelled parent")
	}
}
