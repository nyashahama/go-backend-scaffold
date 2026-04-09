//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	dbgen "github.com/nyashahama/go-backend-scaffold/db/gen"
	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/notification"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/database"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/health"
	"github.com/nyashahama/go-backend-scaffold/internal/server"
)

var (
	testPool  *database.Pool
	testRedis *redis.Client
	testLog   *slog.Logger
)

type redisHealthChecker struct {
	client *redis.Client
}

func (r redisHealthChecker) Ping(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}

func TestMain(m *testing.M) {
	ctx := context.Background()

	testLog = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	// Database
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://user:password@localhost:5432/scaffold?sslmode=disable"
	}
	rawPool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		testLog.Error("failed to connect to test database", "error", err)
		os.Exit(1)
	}
	if err := rawPool.Ping(ctx); err != nil {
		testLog.Error("failed to ping test database", "error", err, "database_url", dbURL)
		os.Exit(1)
	}
	testPool = &database.Pool{Pool: rawPool, Q: dbgen.New(rawPool)}
	if err := requireAuthSchema(ctx, rawPool); err != nil {
		testLog.Error("integration database is missing required schema", "error", err)
		os.Exit(1)
	}

	// Redis
	redisURL := os.Getenv("REDIS_URL")
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		testLog.Error("failed to parse redis URL", "error", err)
		os.Exit(1)
	}
	testRedis = redis.NewClient(opts)
	if err := testRedis.Ping(ctx).Err(); err != nil {
		testLog.Error("failed to ping test redis", "error", err, "redis_url", redisURL)
		os.Exit(1)
	}

	code := m.Run()
	rawPool.Close()
	testRedis.Close()
	os.Exit(code)
}

type successEnvelope[T any] struct {
	Data T `json:"data"`
}

type errorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func decodeSuccess[T any](t *testing.T, recorder *httptest.ResponseRecorder) T {
	t.Helper()
	var envelope successEnvelope[T]
	if err := json.NewDecoder(recorder.Body).Decode(&envelope); err != nil {
		t.Fatalf("decode success envelope: %v", err)
	}
	return envelope.Data
}

func resetRouterState(t *testing.T) {
	t.Helper()

	if _, err := testPool.Exec(context.Background(), `
		TRUNCATE TABLE refresh_tokens, org_memberships, orgs, users CASCADE
	`); err != nil {
		t.Fatalf("truncate auth tables: %v", err)
	}

	if err := testRedis.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}

func newAuthRouter(t *testing.T, sender notification.Sender) http.Handler {
	t.Helper()

	if sender == nil {
		sender = &notification.NoopSender{}
	}

	svc := auth.NewService(
		testPool, testRedis, sender,
		testJWTSigningKey, "http://localhost:3000",
		15*time.Minute, 7*24*time.Hour,
	)

	cfg := &config.Config{
		Env:            "test",
		JWTSecret:      testJWTSigningKey,
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTExpiry:      15 * time.Minute,
		RefreshExpiry:  7 * 24 * time.Hour,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return server.NewRouter(cfg, logger, testPool.Q, testRedis, server.Handlers{
		Health: health.New(nil, nil),
		Auth:   auth.NewHandler(svc),
	})
}

func newStartupRouter(t *testing.T) http.Handler {
	t.Helper()

	svc := auth.NewService(
		testPool, testRedis, &notification.NoopSender{},
		testJWTSigningKey, "http://localhost:3000",
		15*time.Minute, 7*24*time.Hour,
	)

	cfg := &config.Config{
		Env:            "test",
		JWTSecret:      testJWTSigningKey,
		AllowedOrigins: []string{"http://localhost:3000"},
		JWTExpiry:      15 * time.Minute,
		RefreshExpiry:  7 * 24 * time.Hour,
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	return server.NewRouter(cfg, logger, testPool.Q, testRedis, server.Handlers{
		Health: health.New(testPool, redisHealthChecker{client: testRedis}),
		Auth:   auth.NewHandler(svc),
	})
}

func jsonReader(t *testing.T, payload map[string]string) *bytes.Reader {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	return bytes.NewReader(body)
}

func registerViaRouter(t *testing.T, router http.Handler, email, password string) auth.AuthResponse {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", jsonReader(t, map[string]string{
		"email":     email,
		"password":  password,
		"full_name": "Security Test User",
	}))
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", w.Code, w.Body.String())
	}
	return decodeSuccess[auth.AuthResponse](t, w)
}

func authRequest(method, path, accessToken string, body io.Reader) *http.Request {
	req := httptest.NewRequest(method, path, body)
	if accessToken != "" {
		req.Header.Set("Authorization", "Bearer "+accessToken)
	}
	return req
}

func withAuthContext(r *http.Request, accessToken, jwtSecret string) *http.Request {
	claims, err := auth.ValidateAccessToken(accessToken, jwtSecret)
	if err != nil {
		panic("withAuthContext: invalid token: " + err.Error())
	}
	return r.WithContext(auth.ContextWithClaims(r.Context(), claims))
}

func requireAuthSchema(ctx context.Context, pool *pgxpool.Pool) error {
	var usersTable *string
	if err := pool.QueryRow(ctx, "select to_regclass('public.users')::text").Scan(&usersTable); err != nil {
		return err
	}
	if usersTable == nil || *usersTable == "" {
		return errors.New("users table not found; run migrations before integration tests")
	}
	return nil
}
