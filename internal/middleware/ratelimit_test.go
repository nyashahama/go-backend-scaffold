package middleware

import (
	"bytes"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestClientIP_IgnoresForwardedHeadersByDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.5")

	ip := clientIP(req, ClientIPOptions{})

	if ip != "10.0.0.5" {
		t.Fatalf("ip=%q, want 10.0.0.5", ip)
	}
}

func TestClientIP_UsesForwardedHeadersFromTrustedProxy(t *testing.T) {
	_, network, err := net.ParseCIDR("10.0.0.0/8")
	if err != nil {
		t.Fatalf("parse cidr: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:12345"
	req.Header.Set("X-Forwarded-For", "203.0.113.10, 10.0.0.5")

	ip := clientIP(req, ClientIPOptions{
		TrustProxyHeaders: true,
		TrustedProxies:    []*net.IPNet{network},
	})

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

	handler := RateLimit(rdb, RateLimitOptions{})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

	if got := clientIP(req, ClientIPOptions{}); got != "203.0.113.7" {
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

func TestRateLimit_UsesEmailScopedKeyForLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"A.User@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.7:1234"

	key, ok := rateLimitKey(req, ClientIPOptions{}, RateLimitPolicy{
		Scope: RateLimitScopeAuthEmailIP,
	})
	if !ok {
		t.Fatal("expected a rate limit key")
	}
	if !strings.Contains(key, "a.user@example.com") {
		t.Fatalf("key=%q, want normalized email to be included", key)
	}

	body, err := io.ReadAll(req.Body)
	if err != nil {
		t.Fatalf("read restored body: %v", err)
	}
	if got := string(body); got != `{"email":"A.User@example.com"}` {
		t.Fatalf("body=%q, want original request body", got)
	}
}

func TestRateLimit_UsesStricterPolicyForLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)

	policy := policyForRequest(req)
	if policy.Limit >= 100 {
		t.Fatalf("login limit=%d, want stricter than global limit", policy.Limit)
	}
	if policy.Scope != RateLimitScopeAuthEmailIP {
		t.Fatalf("scope=%q, want %q", policy.Scope, RateLimitScopeAuthEmailIP)
	}
}

func TestRateLimitKey_FallsBackToIPWhenEmailMissing(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewBufferString(`{"password":"secret"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.7:1234"

	key, ok := rateLimitKey(req, ClientIPOptions{}, RateLimitPolicy{
		Scope: RateLimitScopeAuthEmailIP,
	})
	if !ok {
		t.Fatal("expected a rate limit key")
	}
	if strings.Contains(key, "email:") {
		t.Fatalf("key=%q, did not expect email scope when email is missing", key)
	}
	if !strings.Contains(key, "ip:198.51.100.7") {
		t.Fatalf("key=%q, want fallback ip scope", key)
	}
}
