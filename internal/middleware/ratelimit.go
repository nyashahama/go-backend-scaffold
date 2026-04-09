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

func RateLimit(rdb *redis.Client, limit int, window time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if rdb == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := clientIP(r)
			key := fmt.Sprintf("ratelimit:%s", ip)
			ctx := r.Context()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				response.Error(w, http.StatusServiceUnavailable, response.CodeInternalError, "rate limiter unavailable")
				return
			}

			if count == 1 {
				if err := rdb.Expire(ctx, key, window).Err(); err != nil {
					response.Error(w, http.StatusServiceUnavailable, response.CodeInternalError, "rate limiter unavailable")
					return
				}
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
