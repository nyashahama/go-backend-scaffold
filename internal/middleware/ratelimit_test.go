package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestClientIP_UsesForwardedForWhenProxyIsLoopback(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 127.0.0.1")

	ip := clientIP(req)

	if ip != "203.0.113.10" {
		t.Fatalf("ip=%q, want 203.0.113.10", ip)
	}
}

func TestRateLimit_FailsClosedWhenRedisUnavailable(t *testing.T) {
	rdb := redis.NewClient(&redis.Options{
		Addr:         "127.0.0.1:1",
		DialTimeout:  20 * time.Millisecond,
		ReadTimeout:  20 * time.Millisecond,
		WriteTimeout: 20 * time.Millisecond,
	})
	defer func() {
		if err := rdb.Close(); err != nil {
			t.Fatalf("close redis client: %v", err)
		}
	}()

	handler := RateLimit(rdb, 100, time.Minute)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "127.0.0.1:54321"
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d, want 503", w.Code)
	}
}

func TestClientIP_FallsBackToRemoteAddr(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "203.0.113.7:4567"

	if got := clientIP(req); got != "203.0.113.7" {
		t.Fatalf("client IP = %q, want %q", got, "203.0.113.7")
	}
}

func TestShouldSkipRateLimit_AllowsInfraEndpoints(t *testing.T) {
	for _, path := range []string{"/healthz", "/readyz", "/metrics"} {
		if !shouldSkipRateLimit(path) {
			t.Fatalf("expected %s to bypass rate limiting", path)
		}
	}
}

func TestShouldSkipRateLimit_LeavesAPIRoutesProtected(t *testing.T) {
	if shouldSkipRateLimit("/api/v1/auth/login") {
		t.Fatal("expected api route to remain rate-limited")
	}
}
