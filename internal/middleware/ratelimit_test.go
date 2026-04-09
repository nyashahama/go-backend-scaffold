package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestClientIP_UsesForwardedForFirstAddress(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.9:4567"
	req.Header.Set("X-Forwarded-For", "203.0.113.5, 10.0.0.9")

	if got := clientIP(req); got != "203.0.113.5" {
		t.Fatalf("client IP = %q, want %q", got, "203.0.113.5")
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
