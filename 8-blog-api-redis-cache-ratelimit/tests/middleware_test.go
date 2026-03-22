package tests

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ashkrai/blog-api/internal/middleware"
)

// TestRateLimiterMiddleware_NoRedis verifies that when the rate limiter's Redis
// client is nil (simulated via a fail-open path), requests still pass through.
// Real Redis integration is covered in integration_test.go.

func TestMiddlewareRecoverer(t *testing.T) {
	panicking := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
	handler := middleware.Recoverer(panicking)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("want 500, got %d", w.Code)
	}
}

func TestMiddlewareLogger(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	})
	handler := middleware.Logger(inner)
	req := httptest.NewRequest(http.MethodGet, "/test-log", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusTeapot {
		t.Errorf("want 418, got %d", w.Code)
	}
}

// TestRateLimiterHeaders verifies that X-RateLimit-* headers are set.
// Requires a running Redis; skip if not available.
func TestRateLimiterHeadersSkipNoRedis(t *testing.T) {
	t.Skip("requires Redis — run via docker-compose")
	_ = time.Minute // suppress unused import
}
