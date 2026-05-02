package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
)

func TestRequireRoleAllowsAllowedRole(t *testing.T) {
	handler := RequireRole(auth.RoleOwner, auth.RoleAdmin)(okHandler())
	req := requestWithIdentity("user-123", "org-123", string(auth.RoleAdmin))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204", w.Code)
	}
}

func TestRequireRoleRejectsMissingIdentity(t *testing.T) {
	handler := RequireRole(auth.RoleAdmin)(okHandler())
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
}

func TestRequireRoleRejectsDisallowedRole(t *testing.T) {
	handler := RequireRole(auth.RoleOwner, auth.RoleAdmin)(okHandler())
	req := requestWithIdentity("user-123", "org-123", string(auth.RoleMember))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", w.Code)
	}
}

func TestRequireOrgAccessAllowsSameOrg(t *testing.T) {
	handler := RequireOrgAccess(func(r *http.Request) (string, bool) {
		return "org-123", true
	})(okHandler())
	req := requestWithIdentity("user-123", "org-123", string(auth.RoleMember))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status=%d, want 204", w.Code)
	}
}

func TestRequireOrgAccessRejectsDifferentOrg(t *testing.T) {
	handler := RequireOrgAccess(func(r *http.Request) (string, bool) {
		return "org-456", true
	})(okHandler())
	req := requestWithIdentity("user-123", "org-123", string(auth.RoleMember))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", w.Code)
	}
}

func TestRequireOrgAccessRejectsMissingOrgID(t *testing.T) {
	handler := RequireOrgAccess(func(r *http.Request) (string, bool) {
		return "", false
	})(okHandler())
	req := requestWithIdentity("user-123", "org-123", string(auth.RoleMember))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", w.Code)
	}
}

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})
}

func requestWithIdentity(userID, orgID, role string) *http.Request {
	req := httptest.NewRequest(http.MethodGet, "/resource", nil)
	return req.WithContext(auth.ContextWithIdentity(req.Context(), userID, orgID, role))
}
