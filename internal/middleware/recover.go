package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

func Recover(env string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					attrs := []any{
						"error", err,
						"method", r.Method,
						"path", r.URL.Path,
					}
					if !isProductionEnv(env) {
						attrs = append(attrs, "stack", string(debug.Stack()))
					}
					slog.Error("panic recovered", attrs...)
					response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func isProductionEnv(env string) bool {
	return strings.EqualFold(env, "production")
}
