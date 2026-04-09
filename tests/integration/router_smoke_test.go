//go:build integration

package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
)

func TestRouter_StartupSmokeFlow(t *testing.T) {
	resetRouterState(t)
	router := newStartupRouter(t)

	register := registerViaRouter(t, router, uniqueEmail(t), testPassword)

	meReq := authRequest(http.MethodGet, "/api/v1/auth/me", register.AccessToken, nil)
	meW := httptest.NewRecorder()
	router.ServeHTTP(meW, meReq)
	if meW.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", meW.Code, meW.Body.String())
	}

	me := decodeSuccess[auth.MeResponse](t, meW)
	if me.ID != register.User.ID {
		t.Fatalf("me user id=%q, want %q", me.ID, register.User.ID)
	}
	if me.Email != register.User.Email {
		t.Fatalf("me email=%q, want %q", me.Email, register.User.Email)
	}
	if me.OrgID == "" {
		t.Fatal("me org_id was empty")
	}

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", jsonReader(t, map[string]string{
		"refresh_token": register.RefreshToken,
	}))
	refreshW := httptest.NewRecorder()
	router.ServeHTTP(refreshW, refreshReq)
	if refreshW.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshW.Code, refreshW.Body.String())
	}

	refresh := decodeSuccess[auth.RefreshResponse](t, refreshW)
	if refresh.AccessToken == "" {
		t.Fatal("refresh access token was empty")
	}
	if refresh.RefreshToken == "" {
		t.Fatal("refresh refresh token was empty")
	}
	if refresh.RefreshToken == register.RefreshToken {
		t.Fatal("refresh token was not rotated")
	}

	readyReq := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	readyW := httptest.NewRecorder()
	router.ServeHTTP(readyW, readyReq)
	if readyW.Code != http.StatusOK {
		t.Fatalf("readyz status=%d body=%s", readyW.Code, readyW.Body.String())
	}
}
