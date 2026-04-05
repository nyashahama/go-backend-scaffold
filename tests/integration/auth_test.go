//go:build integration

package integration

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/notification"
)

const (
	testJWTSigningKey = "for-integration-tests-only"
	testPassword      = "Tr0ub4dor&3-test-only"
)

func newAuthHandler(t *testing.T) *auth.Handler {
	t.Helper()
	svc := auth.NewService(
		testPool, testRedis, &notification.NoopSender{},
		testJWTSigningKey, "http://localhost:3000",
		15*time.Minute, 7*24*time.Hour,
	)
	return auth.NewHandler(svc)
}

func uniqueEmail(t *testing.T) string {
	safe := strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	b := make([]byte, 4)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%s-%x@test.example.com", safe, b)
}

func TestAuth_RegisterLoginRefreshLogout(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// --- Register ---
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Integration User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status=%d body=%s", w.Code, w.Body)
	}
	regResp := decodeSuccess[auth.AuthResponse](t, w)
	if regResp.AccessToken == "" {
		t.Fatal("register: missing access_token")
	}
	if regResp.RefreshToken == "" {
		t.Fatal("register: missing refresh_token")
	}

	// --- Duplicate register → 409 ---
	req = httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w = httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("duplicate register: status=%d, want 409", w.Code)
	}

	// --- Login ---
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": testPassword})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("login: status=%d body=%s", w.Code, w.Body)
	}
	loginResp := decodeSuccess[auth.AuthResponse](t, w)
	originalRefreshToken := loginResp.RefreshToken

	// --- Refresh ---
	refreshBody, _ := json.Marshal(map[string]string{"refresh_token": originalRefreshToken})
	req = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(refreshBody))
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("refresh: status=%d body=%s", w.Code, w.Body)
	}
	refreshResp := decodeSuccess[auth.RefreshResponse](t, w)
	newRefreshToken := refreshResp.RefreshToken
	if newRefreshToken == originalRefreshToken {
		t.Error("refresh: expected rotated token, got same value")
	}

	// --- Old refresh token must be revoked ---
	req = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(refreshBody)) // original token
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("reuse old refresh: status=%d, want 401", w.Code)
	}

	// --- Me ---
	req = httptest.NewRequest(http.MethodGet, "/me", nil)
	req = withAuthContext(req, loginResp.AccessToken, testJWTSigningKey)
	w = httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("me: status=%d body=%s", w.Code, w.Body)
	}
	meResp := decodeSuccess[auth.MeResponse](t, w)
	if meResp.Email != email {
		t.Errorf("me: email = %q, want %q", meResp.Email, email)
	}
	if meResp.Role != "admin" {
		t.Errorf("me: role = %q, want admin", meResp.Role)
	}

	// --- Logout ---
	logoutBody, _ := json.Marshal(map[string]string{"refresh_token": newRefreshToken})
	req = httptest.NewRequest(http.MethodPost, "/logout", bytes.NewReader(logoutBody))
	w = httptest.NewRecorder()
	h.Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("logout: status=%d, want 204", w.Code)
	}

	// --- Refresh after logout must fail ---
	refreshBody2, _ := json.Marshal(map[string]string{"refresh_token": newRefreshToken})
	req = httptest.NewRequest(http.MethodPost, "/refresh", bytes.NewReader(refreshBody2))
	w = httptest.NewRecorder()
	h.Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("refresh after logout: status=%d, want 401", w.Code)
	}
}

func TestAuth_LoginWrongPassword(t *testing.T) {
	h := newAuthHandler(t)
	email := uniqueEmail(t)

	// Register first
	regBody, _ := json.Marshal(map[string]string{
		"email": email, "password": testPassword, "full_name": "Test User",
	})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewReader(regBody))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register: status=%d", w.Code)
	}

	// Login with wrong password
	loginBody, _ := json.Marshal(map[string]string{"email": email, "password": "wrong-password"})
	req = httptest.NewRequest(http.MethodPost, "/login", bytes.NewReader(loginBody))
	w = httptest.NewRecorder()
	h.Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("wrong password login: status=%d, want 401", w.Code)
	}
}
