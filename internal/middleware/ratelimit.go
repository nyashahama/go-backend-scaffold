package middleware

import (
	"fmt"
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

func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if shouldSkipRateLimit(r.URL.Path) || rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			key := fmt.Sprintf("ratelimit:%s", ip)
			count, err := rateLimitScript.Run(r.Context(), rdb, []string{key}, window.Milliseconds()).Int64()
			if err != nil {
				response.Error(w, http.StatusServiceUnavailable, response.CodeInternalError, "rate limiter unavailable")
				return
			}

			if count > int64(limit) {
				response.Error(w, http.StatusTooManyRequests, response.CodeRateLimited, "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		host = r.RemoteAddr
	}

	ip := net.ParseIP(host)
	if ip != nil && (ip.IsLoopback() || ip.IsPrivate()) {
		if forwarded := firstForwardedIP(r.Header.Get("X-Forwarded-For")); forwarded != "" {
			return forwarded
		}
		if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" && net.ParseIP(realIP) != nil {
			return realIP
		}
	}

	return host
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

func shouldSkipRateLimit(path string) bool {
	switch path {
	case "/healthz", "/readyz", "/metrics":
		return true
	default:
		return false
	}
}
