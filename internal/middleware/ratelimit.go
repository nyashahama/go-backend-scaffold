package middleware

import (
	"context"
	"fmt"
	"net/http"
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

			ip := r.RemoteAddr
			key := fmt.Sprintf("ratelimit:%s", ip)
			ctx := context.Background()

			count, err := rdb.Incr(ctx, key).Result()
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			if count == 1 {
				rdb.Expire(ctx, key, window)
			}

			if count > int64(limit) {
				response.Error(w, http.StatusTooManyRequests, response.CodeRateLimited, "too many requests")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
