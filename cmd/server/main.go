package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/notification"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/cache"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/database"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/health"
	"github.com/nyashahama/go-backend-scaffold/internal/server"
)

// redisChecker adapts *redis.Client to health.Checker.
type redisChecker struct{ c *redis.Client }

func (rc *redisChecker) Ping(ctx context.Context) error {
	return rc.c.Ping(ctx).Err()
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	// Load .env in development (no-op if file doesn't exist or vars already set)
	_ = godotenv.Load()

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	ctx := context.Background()

	// Database
	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("database connected")

	// Redis
	rdb, err := cache.New(ctx, cfg.RedisURL)
	if err != nil {
		logger.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	defer rdb.Close()
	logger.Info("redis connected")

	// Auth service — swap &notification.NoopSender{} for a real Sender per project
	authService := auth.NewService(
		db, rdb, &notification.NoopSender{},
		cfg.JWTSecret, cfg.AppBaseURL,
		cfg.JWTExpiry, cfg.RefreshExpiry,
	)

	handlers := server.Handlers{
		Health: health.New(db, &redisChecker{rdb}),
		Auth:   auth.NewHandler(authService),
	}

	router := server.NewRouter(cfg, logger, rdb, handlers)
	srv := server.New(router, cfg.Port, logger)

	logger.Info("starting server", "port", cfg.Port, "env", cfg.Env)
	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}
