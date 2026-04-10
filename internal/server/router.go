package server

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/middleware"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/health"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/response"
)

// Handlers holds all domain handlers. Add new domains here as your project grows.
type Handlers struct {
	Health *health.Handler
	Auth   *auth.Handler
}

func NewRouter(cfg *config.Config, logger *slog.Logger, users middleware.UserReader, rdb *redis.Client, h Handlers) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack (order matters)
	r.Use(middleware.Recover(cfg.Env))
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, middleware.RateLimitOptions{
		ClientIP: middleware.ClientIPOptions{
			TrustProxyHeaders: cfg.TrustProxyHeaders,
			TrustedProxies:    cfg.TrustedProxyCIDRs,
		},
	}))

	// Infrastructure endpoints (no auth, no versioning)
	r.Get("/healthz", h.Health.Healthz)
	r.Get("/readyz", h.Health.Readyz)
	if cfg.Env != "production" {
		r.Handle("/metrics", promhttp.Handler())
	} else if cfg.MetricsBearerToken != "" {
		r.With(metricsBearerAuth(cfg.MetricsBearerToken)).Handle("/metrics", promhttp.Handler())
	}

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		r.Route("/auth", func(r chi.Router) {
			h.Auth.RegisterRoutes(r, middleware.Auth(cfg.JWTSecret, users))
		})
	})

	return r
}

func metricsBearerAuth(token string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Header.Get("Authorization") != "Bearer "+token {
				response.Error(w, http.StatusUnauthorized, response.CodeUnauthorized, "missing or invalid metrics token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
