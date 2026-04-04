package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("panic recovered",
					"error", err,
					"stack", string(debug.Stack()),
					"method", r.Method,
					"path", r.URL.Path,
				)
				response.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "an unexpected error occurred")
			}
		}()
		next.ServeHTTP(w, r)
	})
}
