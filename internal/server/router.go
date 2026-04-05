package server

import (
	"log/slog"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/middleware"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/health"
)

// Handlers holds all domain handlers. Add new domains here as your project grows.
type Handlers struct {
	Health *health.Handler
	Auth   *auth.Handler
}

func NewRouter(cfg *config.Config, logger *slog.Logger, rdb *redis.Client, h Handlers) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware stack (order matters)
	r.Use(middleware.Recover)
	r.Use(middleware.Metrics)
	r.Use(middleware.Logger(logger))
	r.Use(middleware.CORS(cfg.AllowedOrigins))
	r.Use(middleware.RateLimit(rdb, 100, 1*time.Minute))

	// Infrastructure endpoints (no auth, no versioning)
	r.Get("/healthz", h.Health.Healthz)
	r.Get("/readyz", h.Health.Readyz)
	r.Handle("/metrics", promhttp.Handler())

	// API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes — no JWT required
		r.Mount("/auth", h.Auth.Routes())

		// Protected routes — JWT required
		r.Group(func(r chi.Router) {
			r.Use(middleware.Auth(cfg.JWTSecret))
			r.Mount("/auth", h.Auth.ProtectedRoutes())
		})
	})

	return r
}
