package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

// mockService is a test double for Servicer.
type mockService struct {
	registerFn       func(ctx context.Context, email, password, fullName string) (*AuthResponse, error)
	loginFn          func(ctx context.Context, email, password, orgID string) (*AuthResponse, error)
	refreshFn        func(ctx context.Context, refreshToken string) (*RefreshResponse, error)
	logoutFn         func(ctx context.Context, refreshToken string) error
	meFn             func(ctx context.Context, userID, orgID string) (*MeResponse, error)
	forgotFn         func(ctx context.Context, email string) error
	resetFn          func(ctx context.Context, token, password string) error
	changePasswordFn func(ctx context.Context, userID, currentPassword, nextPassword string) error
}

func (m *mockService) Register(ctx context.Context, email, password, fullName string) (*AuthResponse, error) {
	if m.registerFn == nil {
		return nil, nil
	}
	return m.registerFn(ctx, email, password, fullName)
}
func (m *mockService) Login(ctx context.Context, email, password, orgID string) (*AuthResponse, error) {
	if m.loginFn == nil {
		return nil, nil
	}
	return m.loginFn(ctx, email, password, orgID)
}
func (m *mockService) Refresh(ctx context.Context, refreshToken string) (*RefreshResponse, error) {
	if m.refreshFn == nil {
		return nil, nil
	}
	return m.refreshFn(ctx, refreshToken)
}
func (m *mockService) Logout(ctx context.Context, refreshToken string) error {
	if m.logoutFn == nil {
		return nil
	}
	return m.logoutFn(ctx, refreshToken)
}
func (m *mockService) Me(ctx context.Context, userID, orgID string) (*MeResponse, error) {
	if m.meFn == nil {
		return nil, nil
	}
	return m.meFn(ctx, userID, orgID)
}
func (m *mockService) ForgotPassword(ctx context.Context, email string) error {
	if m.forgotFn == nil {
		return nil
	}
	return m.forgotFn(ctx, email)
}
func (m *mockService) ResetPassword(ctx context.Context, token, password string) error {
	if m.resetFn == nil {
		return nil
	}
	return m.resetFn(ctx, token, password)
}
func (m *mockService) ChangePassword(ctx context.Context, userID, currentPassword, nextPassword string) error {
	if m.changePasswordFn == nil {
		return nil
	}
	return m.changePasswordFn(ctx, userID, currentPassword, nextPassword)
}

// helpers

func jsonBody(t *testing.T, m map[string]string) *bytes.Reader {
	t.Helper()
	b, _ := json.Marshal(m)
	return bytes.NewReader(b)
}

// --- Register ---

func TestRegister_BadJSON(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/register", bytes.NewBufferString("not-json"))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRegister_MissingFields(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{"email": "a@b.com"}))
	w := httptest.NewRecorder()
	h.Register(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestRegister_DuplicateEmail(t *testing.T) {
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			return nil, ErrEmailExists
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{
		"email": "a@b.com", "password": "pass", "full_name": "A B",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409", w.Code)
	}
}

func TestRegister_Success(t *testing.T) {
	svc := &mockService{
		registerFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			return &AuthResponse{AccessToken: "tok", RefreshToken: "rt", ExpiresIn: 900}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/register", jsonBody(t, map[string]string{
		"email": "a@b.com", "password": "pass", "full_name": "A B",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Register(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", w.Code)
	}
}

// --- Login ---

func TestLogin_InvalidCredentials(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			return nil, ErrInvalidCredentials
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", jsonBody(t, map[string]string{
		"email": "a@b.com", "password": "wrong",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestLogin_Success(t *testing.T) {
	svc := &mockService{
		loginFn: func(_ context.Context, _, _, _ string) (*AuthResponse, error) {
			return &AuthResponse{AccessToken: "tok", RefreshToken: "rt", ExpiresIn: 900}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/login", jsonBody(t, map[string]string{
		"email": "a@b.com", "password": "correct",
	}))
	w := httptest.NewRecorder()
	NewHandler(svc).Login(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// --- Refresh ---

func TestRefresh_InvalidToken(t *testing.T) {
	svc := &mockService{
		refreshFn: func(_ context.Context, _ string) (*RefreshResponse, error) {
			return nil, ErrInvalidToken
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/refresh", jsonBody(t, map[string]string{"refresh_token": "bad"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Refresh(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestRefresh_Success(t *testing.T) {
	svc := &mockService{
		refreshFn: func(_ context.Context, _ string) (*RefreshResponse, error) {
			return &RefreshResponse{AccessToken: "new", RefreshToken: "new-rt", ExpiresIn: 900}, nil
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/refresh", jsonBody(t, map[string]string{"refresh_token": "old"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Refresh(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
}

// --- Logout ---

func TestLogout_NoContent(t *testing.T) {
	svc := &mockService{logoutFn: func(_ context.Context, _ string) error { return nil }}
	req := httptest.NewRequest(http.MethodPost, "/logout", jsonBody(t, map[string]string{"refresh_token": "tok"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204", w.Code)
	}
}

func TestLogout_Idempotent(t *testing.T) {
	svc := &mockService{logoutFn: func(_ context.Context, _ string) error {
		return errors.New("not found")
	}}
	req := httptest.NewRequest(http.MethodPost, "/logout", jsonBody(t, map[string]string{"refresh_token": "gone"}))
	w := httptest.NewRecorder()
	NewHandler(svc).Logout(w, req)
	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want 204 (idempotent)", w.Code)
	}
}

// --- Me ---

func TestMe_MissingContext(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	w := httptest.NewRecorder()
	h.Me(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestMe_Success(t *testing.T) {
	svc := &mockService{
		meFn: func(_ context.Context, _, _ string) (*MeResponse, error) {
			return &MeResponse{
				ID: "u1", Email: "a@b.com", FullName: "A B",
				OrgID: "o1", OrgName: "Org", Role: "admin",
			}, nil
		},
	}
	req := httptest.NewRequest(http.MethodGet, "/me", nil)
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()
	NewHandler(svc).Me(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp struct {
		Data MeResponse `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Role != "admin" {
		t.Errorf("role = %q, want admin", resp.Data.Role)
	}
}

// --- ChangePassword ---

func TestChangePassword_WrongPassword(t *testing.T) {
	svc := &mockService{
		changePasswordFn: func(_ context.Context, _, _, _ string) error {
			return ErrWrongPassword
		},
	}
	req := httptest.NewRequest(http.MethodPost, "/change-password", jsonBody(t, map[string]string{
		"current_password": "wrong", "new_password": "new",
	}))
	req = req.WithContext(ContextWithIdentity(req.Context(), "u1", "o1", string(RoleAdmin)))
	w := httptest.NewRecorder()
	NewHandler(svc).ChangePassword(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}

func TestChangePassword_MissingContext(t *testing.T) {
	h := NewHandler(&mockService{})
	req := httptest.NewRequest(http.MethodPost, "/change-password", jsonBody(t, map[string]string{
		"current_password": "old", "new_password": "new",
	}))
	w := httptest.NewRecorder()
	h.ChangePassword(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401", w.Code)
	}
}
