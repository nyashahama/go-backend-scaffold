package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

var rateLimitScript = redis.NewScript(`
local count = redis.call("INCR", KEYS[1])
if count == 1 then
  redis.call("PEXPIRE", KEYS[1], ARGV[1])
end
return count
`)

type ClientIPOptions struct {
	TrustProxyHeaders bool
	TrustedProxies    []*net.IPNet
}

type RateLimitScope string

const (
	RateLimitScopeIP          RateLimitScope = "ip"
	RateLimitScopeAuthEmailIP RateLimitScope = "auth_email_ip"
)

type RateLimitPolicy struct {
	Name   string
	Limit  int
	Window time.Duration
	Scope  RateLimitScope
}

type RateLimitOptions struct {
	ClientIP ClientIPOptions
}

func RateLimit(rdb *redis.Client, opts RateLimitOptions) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipRateLimit(r.URL.Path) || rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			policy := policyForRequest(r)
			key, ok := rateLimitKey(r, opts.ClientIP, policy)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			count, err := rateLimitScript.Run(r.Context(), rdb, []string{key}, policy.Window.Milliseconds()).Int64()
			if err != nil {
				response.Error(w, http.StatusServiceUnavailable, response.CodeInternalError, "rate limiter unavailable")
				return
			}

			if count > int64(policy.Limit) {
				response.Error(w, http.StatusTooManyRequests, response.CodeRateLimited, "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func policyForRequest(r *http.Request) RateLimitPolicy {
	switch {
	case r.URL.Path == "/api/v1/auth/login":
		return RateLimitPolicy{Name: "auth-login", Limit: 10, Window: time.Minute, Scope: RateLimitScopeAuthEmailIP}
	case r.URL.Path == "/api/v1/auth/forgot-password":
		return RateLimitPolicy{Name: "auth-forgot-password", Limit: 5, Window: 15 * time.Minute, Scope: RateLimitScopeAuthEmailIP}
	case r.URL.Path == "/api/v1/auth/reset-password":
		return RateLimitPolicy{Name: "auth-reset-password", Limit: 10, Window: 15 * time.Minute, Scope: RateLimitScopeIP}
	case r.URL.Path == "/api/v1/auth/refresh":
		return RateLimitPolicy{Name: "auth-refresh", Limit: 20, Window: time.Minute, Scope: RateLimitScopeIP}
	case r.URL.Path == "/api/v1/auth/register":
		return RateLimitPolicy{Name: "auth-register", Limit: 10, Window: 15 * time.Minute, Scope: RateLimitScopeAuthEmailIP}
	default:
		return RateLimitPolicy{Name: "default", Limit: 100, Window: time.Minute, Scope: RateLimitScopeIP}
	}
}

func rateLimitKey(r *http.Request, clientOpts ClientIPOptions, policy RateLimitPolicy) (string, bool) {
	ip := clientIP(r, clientOpts)
	if ip == "" {
		return "", false
	}

	switch policy.Scope {
	case RateLimitScopeAuthEmailIP:
		email := requestEmail(r)
		if email == "" {
			return fmt.Sprintf("ratelimit:%s:ip:%s", policy.Name, ip), true
		}
		return fmt.Sprintf("ratelimit:%s:ip:%s:email:%s", policy.Name, ip, email), true
	default:
		return fmt.Sprintf("ratelimit:%s:ip:%s", policy.Name, ip), true
	}
}

func clientIP(r *http.Request, opts ClientIPOptions) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip != nil && opts.TrustProxyHeaders && isTrustedProxy(ip, opts.TrustedProxies) {
		if forwarded := firstForwardedIP(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			return forwarded
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" && net.ParseIP(realIP) != nil {
			return realIP
		}
	}

	return host
}

func isTrustedProxy(ip net.IP, trusted []*net.IPNet) bool {
	for _, network := range trusted {
		if network != nil && network.Contains(ip) {
			return true
		}
	}
	return false
}

func firstForwardedIP(xff string) string {
	if xff == "" {
		return ""
	}

	for _, part := range strings.Split(xff, ",") {
		candidate := strings.TrimSpace(part)
		if candidate == "" {
			continue
		}
		if net.ParseIP(candidate) != nil {
			return candidate
		}
	}

	return ""
}

func requestEmail(r *http.Request) string {
	if r.Body == nil {
		return ""
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		r.Body = io.NopCloser(bytes.NewReader(nil))
		return ""
	}
	r.Body = io.NopCloser(bytes.NewReader(body))

	var payload struct {
		Email string `json:"email"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}

	return normalizeRateLimitEmail(payload.Email)
}

func normalizeRateLimitEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	return email
}

func shouldSkipRateLimit(path string) bool {
	switch path {
	case "/healthz", "/readyz", "/metrics":
		return true
	default:
		return false
	}
}
