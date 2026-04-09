package server

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/health"
)

func TestNewRouter_DoesNotPanicWhenAuthRoutesAreRegistered(t *testing.T) {
	cfg := &config.Config{JWTSecret: "test-secret"}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handlers := Handlers{
		Health: health.New(nil, nil),
		Auth:   auth.NewHandler(nil),
	}

	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("NewRouter panicked: %v", r)
		}
	}()

	_ = NewRouter(cfg, logger, nil, nil, handlers)
}

func TestNewRouter_HidesMetricsInProductionWithoutToken(t *testing.T) {
	cfg := &config.Config{
		Env:       "production",
		JWTSecret: "test-secret-that-is-long-enough",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handlers := Handlers{
		Health: health.New(nil, nil),
		Auth:   auth.NewHandler(nil),
	}

	router := NewRouter(cfg, logger, nil, nil, handlers)
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status=%d, want 404", w.Code)
	}
}

func TestNewRouter_ProtectsMetricsWithBearerToken(t *testing.T) {
	cfg := &config.Config{
		Env:                "production",
		JWTSecret:          "test-secret-that-is-long-enough",
		MetricsBearerToken: "metrics-secret",
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	handlers := Handlers{
		Health: health.New(nil, nil),
		Auth:   auth.NewHandler(nil),
	}

	router := NewRouter(cfg, logger, nil, nil, handlers)

	unauthReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	unauthW := httptest.NewRecorder()
	router.ServeHTTP(unauthW, unauthReq)
	if unauthW.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status=%d, want 401", unauthW.Code)
	}

	authReq := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	authReq.Header.Set("Authorization", "Bearer metrics-secret")
	authW := httptest.NewRecorder()
	router.ServeHTTP(authW, authReq)
	if authW.Code != http.StatusOK {
		t.Fatalf("authorized status=%d, want 200", authW.Code)
	}
}
