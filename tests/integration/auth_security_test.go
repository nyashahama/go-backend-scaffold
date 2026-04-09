//go:build integration

package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"

	dbgen "github.com/nyashahama/go-backend-scaffold/db/gen"
	"github.com/nyashahama/go-backend-scaffold/internal/auth"
	"github.com/nyashahama/go-backend-scaffold/internal/config"
	"github.com/nyashahama/go-backend-scaffold/internal/notification"
	"github.com/nyashahama/go-backend-scaffold/internal/platform/health"
	"github.com/nyashahama/go-backend-scaffold/internal/server"
)

type captureSender struct {
	resetURL string
}

func (c *captureSender) SendPasswordReset(_ context.Context, _, resetURL string) error {
	c.resetURL = resetURL
	return nil
}

func resetAuthState(t *testing.T) {
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

func addOrgMembership(t *testing.T, userID string, role auth.Role) dbgen.Org {
	t.Helper()

	uid, err := uuid.Parse(userID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}

	org, err := testPool.Q.CreateOrg(context.Background(), "Second Org")
	if err != nil {
		t.Fatalf("create org: %v", err)
	}

	_, err = testPool.Q.CreateOrgMembership(context.Background(), dbgen.CreateOrgMembershipParams{
		UserID: uid,
		OrgID:  org.ID,
		Role:   string(role),
	})
	if err != nil {
		t.Fatalf("create org membership: %v", err)
	}

	return org
}

func TestAuth_RegisterRejectsWeakPassword(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", jsonReader(t, map[string]string{
		"email":     uniqueEmail(t),
		"password":  "weak",
		"full_name": "Weak Password User",
	}))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", w.Code, w.Body.String())
	}
}

func TestAuth_RegisterNormalizesEmailForUniqueness(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	lower := uniqueEmail(t)
	upper := strings.ToUpper(lower)

	first := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", jsonReader(t, map[string]string{
		"email":     upper,
		"password":  testPassword,
		"full_name": "Upper Email",
	}))
	firstW := httptest.NewRecorder()
	router.ServeHTTP(firstW, first)
	if firstW.Code != http.StatusCreated {
		t.Fatalf("first register status=%d body=%s", firstW.Code, firstW.Body.String())
	}

	second := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", jsonReader(t, map[string]string{
		"email":     lower,
		"password":  testPassword,
		"full_name": "Lower Email",
	}))
	secondW := httptest.NewRecorder()
	router.ServeHTTP(secondW, second)

	if secondW.Code != http.StatusConflict {
		t.Fatalf("status=%d body=%s, want 409", secondW.Code, secondW.Body.String())
	}
}

func TestAuth_LoginStoresHashedRefreshToken(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	reg := registerViaRouter(t, router, uniqueEmail(t), testPassword)
	userID, err := uuid.Parse(reg.User.ID)
	if err != nil {
		t.Fatalf("parse user id: %v", err)
	}

	var storedToken string
	if err := testPool.QueryRow(context.Background(),
		`SELECT token FROM refresh_tokens WHERE user_id = $1 ORDER BY created_at DESC LIMIT 1`,
		userID,
	).Scan(&storedToken); err != nil {
		t.Fatalf("load refresh token: %v", err)
	}

	if storedToken == reg.RefreshToken {
		t.Fatalf("refresh token was stored in plaintext")
	}
}

func TestAuth_LogoutInvalidatesExistingAccessToken(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	reg := registerViaRouter(t, router, uniqueEmail(t), testPassword)

	meBefore := httptest.NewRecorder()
	router.ServeHTTP(meBefore, authRequest(http.MethodGet, "/api/v1/auth/me", reg.AccessToken, nil))
	if meBefore.Code != http.StatusOK {
		t.Fatalf("me before logout status=%d body=%s", meBefore.Code, meBefore.Body.String())
	}

	logout := httptest.NewRecorder()
	router.ServeHTTP(logout, httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", jsonReader(t, map[string]string{
		"refresh_token": reg.RefreshToken,
	})))
	if logout.Code != http.StatusNoContent {
		t.Fatalf("logout status=%d body=%s", logout.Code, logout.Body.String())
	}

	meAfter := httptest.NewRecorder()
	router.ServeHTTP(meAfter, authRequest(http.MethodGet, "/api/v1/auth/me", reg.AccessToken, nil))

	if meAfter.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", meAfter.Code, meAfter.Body.String())
	}
}

func TestAuth_ChangePasswordInvalidatesExistingAccessToken(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	reg := registerViaRouter(t, router, uniqueEmail(t), testPassword)

	change := httptest.NewRecorder()
	router.ServeHTTP(change, authRequest(http.MethodPost, "/api/v1/auth/change-password", reg.AccessToken, jsonReader(t, map[string]string{
		"current_password": testPassword,
		"new_password":     "StrongerPassw0rd!",
	})))
	if change.Code != http.StatusNoContent {
		t.Fatalf("change password status=%d body=%s", change.Code, change.Body.String())
	}

	meAfter := httptest.NewRecorder()
	router.ServeHTTP(meAfter, authRequest(http.MethodGet, "/api/v1/auth/me", reg.AccessToken, nil))

	if meAfter.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", meAfter.Code, meAfter.Body.String())
	}
}

func TestAuth_ForgotPasswordDoesNotStoreRawResetToken(t *testing.T) {
	resetAuthState(t)
	sender := &captureSender{}
	router := newAuthRouter(t, sender)

	email := uniqueEmail(t)
	registerViaRouter(t, router, email, testPassword)

	forgot := httptest.NewRecorder()
	router.ServeHTTP(forgot, httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", jsonReader(t, map[string]string{
		"email": email,
	})))
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot password status=%d body=%s", forgot.Code, forgot.Body.String())
	}
	if sender.resetURL == "" {
		t.Fatal("expected password reset URL to be captured")
	}

	resetURL, err := url.Parse(sender.resetURL)
	if err != nil {
		t.Fatalf("parse reset url: %v", err)
	}
	token := resetURL.Query().Get("token")
	if token == "" {
		t.Fatal("expected reset token in query string")
	}

	rawKeyExists, err := testRedis.Exists(context.Background(), "pwreset:"+token).Result()
	if err != nil {
		t.Fatalf("check raw redis key: %v", err)
	}
	if rawKeyExists != 0 {
		t.Fatalf("reset token was stored using the raw token value")
	}
}

func TestAuth_ResetPasswordInvalidatesExistingAccessToken(t *testing.T) {
	resetAuthState(t)
	sender := &captureSender{}
	router := newAuthRouter(t, sender)

	email := uniqueEmail(t)
	reg := registerViaRouter(t, router, email, testPassword)

	forgot := httptest.NewRecorder()
	router.ServeHTTP(forgot, httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", jsonReader(t, map[string]string{
		"email": email,
	})))
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot password status=%d body=%s", forgot.Code, forgot.Body.String())
	}

	resetURL, err := url.Parse(sender.resetURL)
	if err != nil {
		t.Fatalf("parse reset url: %v", err)
	}
	token := resetURL.Query().Get("token")
	if token == "" {
		t.Fatal("expected reset token in query string")
	}

	reset := httptest.NewRecorder()
	router.ServeHTTP(reset, httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", jsonReader(t, map[string]string{
		"token":    token,
		"password": "StrongerResetPassw0rd!",
	})))
	if reset.Code != http.StatusNoContent {
		t.Fatalf("reset password status=%d body=%s", reset.Code, reset.Body.String())
	}

	meAfter := httptest.NewRecorder()
	router.ServeHTTP(meAfter, authRequest(http.MethodGet, "/api/v1/auth/me", reg.AccessToken, nil))

	if meAfter.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d body=%s, want 401", meAfter.Code, meAfter.Body.String())
	}
}

func TestAuth_RefreshRejectsConcurrentReuse(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)
	reg := registerViaRouter(t, router, uniqueEmail(t), testPassword)

	const attempts = 6
	start := make(chan struct{})
	results := make(chan int, attempts)
	var wg sync.WaitGroup

	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", jsonReader(t, map[string]string{
				"refresh_token": reg.RefreshToken,
			})))
			results <- w.Code
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var success, unauthorized int
	for code := range results {
		switch code {
		case http.StatusOK:
			success++
		case http.StatusUnauthorized:
			unauthorized++
		default:
			t.Fatalf("unexpected status from concurrent refresh: %d", code)
		}
	}

	if success != 1 {
		t.Fatalf("successes=%d, want 1", success)
	}
	if unauthorized != attempts-1 {
		t.Fatalf("unauthorized=%d, want %d", unauthorized, attempts-1)
	}
}

func TestAuth_ResetPasswordRejectsConcurrentReuse(t *testing.T) {
	resetAuthState(t)
	sender := &captureSender{}
	router := newAuthRouter(t, sender)

	email := uniqueEmail(t)
	registerViaRouter(t, router, email, testPassword)

	forgot := httptest.NewRecorder()
	router.ServeHTTP(forgot, httptest.NewRequest(http.MethodPost, "/api/v1/auth/forgot-password", jsonReader(t, map[string]string{
		"email": email,
	})))
	if forgot.Code != http.StatusOK {
		t.Fatalf("forgot password status=%d body=%s", forgot.Code, forgot.Body.String())
	}

	resetURL, err := url.Parse(sender.resetURL)
	if err != nil {
		t.Fatalf("parse reset url: %v", err)
	}
	token := resetURL.Query().Get("token")
	if token == "" {
		t.Fatal("expected reset token in query string")
	}

	const attempts = 6
	start := make(chan struct{})
	results := make(chan int, attempts)
	var wg sync.WaitGroup

	for range attempts {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start

			w := httptest.NewRecorder()
			router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/reset-password", jsonReader(t, map[string]string{
				"token":    token,
				"password": "ConcurrentResetPassw0rd!",
			})))
			results <- w.Code
		}()
	}

	close(start)
	wg.Wait()
	close(results)

	var success, unauthorized int
	for code := range results {
		switch code {
		case http.StatusNoContent:
			success++
		case http.StatusUnauthorized:
			unauthorized++
		default:
			t.Fatalf("unexpected status from concurrent reset: %d", code)
		}
	}

	if success != 1 {
		t.Fatalf("successes=%d, want 1", success)
	}
	if unauthorized != attempts-1 {
		t.Fatalf("unauthorized=%d, want %d", unauthorized, attempts-1)
	}
}

func TestAuth_LoginRequiresExplicitOrgSelectionForMultiOrgUser(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	email := uniqueEmail(t)
	reg := registerViaRouter(t, router, email, testPassword)
	addOrgMembership(t, reg.User.ID, auth.RoleMember)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", jsonReader(t, map[string]string{
		"email":    email,
		"password": testPassword,
	})))

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d body=%s, want 400", w.Code, w.Body.String())
	}
}

func TestAuth_LoginWithExplicitOrgSelectionUsesRequestedOrg(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	email := uniqueEmail(t)
	reg := registerViaRouter(t, router, email, testPassword)
	secondOrg := addOrgMembership(t, reg.User.ID, auth.RoleMember)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", jsonReader(t, map[string]string{
		"email":    email,
		"password": testPassword,
		"org_id":   secondOrg.ID.String(),
	})))

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}

	loginResp := decodeSuccess[auth.AuthResponse](t, w)
	claims, err := auth.ValidateAccessToken(loginResp.AccessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.OrgID != secondOrg.ID.String() {
		t.Fatalf("org_id=%q, want %q", claims.OrgID, secondOrg.ID.String())
	}
	if claims.Role != string(auth.RoleMember) {
		t.Fatalf("role=%q, want %q", claims.Role, auth.RoleMember)
	}
}

func TestAuth_RefreshPreservesSelectedOrgForMultiOrgUser(t *testing.T) {
	resetAuthState(t)
	router := newAuthRouter(t, nil)

	email := uniqueEmail(t)
	reg := registerViaRouter(t, router, email, testPassword)
	secondOrg := addOrgMembership(t, reg.User.ID, auth.RoleMember)

	loginW := httptest.NewRecorder()
	router.ServeHTTP(loginW, httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", jsonReader(t, map[string]string{
		"email":    email,
		"password": testPassword,
		"org_id":   secondOrg.ID.String(),
	})))
	if loginW.Code != http.StatusOK {
		t.Fatalf("login status=%d body=%s", loginW.Code, loginW.Body.String())
	}
	loginResp := decodeSuccess[auth.AuthResponse](t, loginW)

	refreshW := httptest.NewRecorder()
	router.ServeHTTP(refreshW, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", jsonReader(t, map[string]string{
		"refresh_token": loginResp.RefreshToken,
	})))
	if refreshW.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshW.Code, refreshW.Body.String())
	}

	refreshResp := decodeSuccess[auth.RefreshResponse](t, refreshW)
	claims, err := auth.ValidateAccessToken(refreshResp.AccessToken, testJWTSigningKey)
	if err != nil {
		t.Fatalf("validate access token: %v", err)
	}
	if claims.OrgID != secondOrg.ID.String() {
		t.Fatalf("org_id=%q, want %q", claims.OrgID, secondOrg.ID.String())
	}
	if claims.Role != string(auth.RoleMember) {
		t.Fatalf("role=%q, want %q", claims.Role, auth.RoleMember)
	}
}
