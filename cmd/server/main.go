package main

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"

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
	defer func() { _ = rdb.Close() }()
	logger.Info("redis connected")

	sender, err := buildNotificationSender(cfg)
	if err != nil {
		logger.Error("invalid runtime configuration", "error", err)
		os.Exit(1)
	}
	if err := validateRuntimeConfig(cfg, sender); err != nil {
		logger.Error("invalid runtime configuration", "error", err)
		os.Exit(1)
	}

	authService := auth.NewService(
		db, rdb, sender,
		cfg.JWTSecret, cfg.AppBaseURL,
		cfg.JWTExpiry, cfg.RefreshExpiry,
	)

	handlers := server.Handlers{
		Health: health.New(db, &redisChecker{rdb}),
		Auth:   auth.NewHandler(authService),
	}

	router := server.NewRouter(cfg, logger, db.Q, rdb, handlers)
	srv := server.New(router, cfg.Port, logger)

	if err := srv.Start(); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}

	logger.Info("server stopped")
}

func validateRuntimeConfig(cfg *config.Config, sender notification.Sender) error {
	if cfg == nil {
		return errors.New("config is required")
	}

	if !strings.EqualFold(cfg.Env, "production") {
		return nil
	}

	if cfg.AppBaseURL == "" {
		return errors.New("APP_BASE_URL is required in production")
	}

	if cfg.ResendAPIKey == "" || cfg.EmailFrom == "" || cfg.EmailFromName == "" {
		return errors.New("production requires RESEND_API_KEY, EMAIL_FROM, and EMAIL_FROM_NAME")
	}

	if _, ok := sender.(*notification.NoopSender); ok {
		return errors.New("production requires a real notification sender")
	}

	return nil
}

func buildNotificationSender(cfg *config.Config) (notification.Sender, error) {
	if cfg == nil {
		return nil, errors.New("config is required")
	}

	hasAnyResendConfig := cfg.ResendAPIKey != "" || cfg.EmailFrom != "" || cfg.EmailFromName != ""
	hasFullResendConfig := cfg.ResendAPIKey != "" && cfg.EmailFrom != "" && cfg.EmailFromName != ""

	if !hasAnyResendConfig {
		return &notification.NoopSender{}, nil
	}
	if !hasFullResendConfig {
		return nil, errors.New("RESEND_API_KEY, EMAIL_FROM, and EMAIL_FROM_NAME must be set together")
	}

	return notification.NewResendSender(cfg.ResendAPIKey, cfg.EmailFrom, cfg.EmailFromName)
}
