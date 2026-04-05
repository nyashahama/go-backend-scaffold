//go:build integration

package integration

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/database"
	dbgen "github.com/nyashahama/go-backend-scaffold/db/gen"
)

var (
	testPool  *database.Pool
	testRedis *redis.Client
	testLog   *slog.Logger
)

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
	testPool = &database.Pool{Pool: rawPool, Q: dbgen.New(rawPool)}
	defer rawPool.Close()

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
	defer testRedis.Close()

	os.Exit(m.Run())
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

func withAuthContext(r *http.Request, accessToken, jwtSecret string) *http.Request {
	claims, err := auth.ValidateAccessToken(accessToken, jwtSecret)
	if err != nil {
		panic("withAuthContext: invalid token: " + err.Error())
	}
	return r.WithContext(auth.ContextWithClaims(r.Context(), claims))
}
