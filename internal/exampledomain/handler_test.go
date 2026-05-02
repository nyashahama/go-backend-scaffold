package exampledomain

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"github.com/nyashahama/go-backend-scaffold/internal/auth"
)

func TestRoutesListResourcesRequiresMatchingOrg(t *testing.T) {
	handler := NewHandler(&fakeService{
		resources: []Resource{
			{ID: "res-1", OrgID: "org-123", Name: "First resource"},
		},
	})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/orgs/org-123/resources", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "user-123", "org-123", string(auth.RoleMember)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}

	var body struct {
		Data []Resource `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(body.Data) != 1 {
		t.Fatalf("len(data)=%d, want 1", len(body.Data))
	}
	if body.Data[0].ID != "res-1" {
		t.Fatalf("resource ID=%q, want res-1", body.Data[0].ID)
	}
}

func TestRoutesListResourcesRejectsCrossOrgAccess(t *testing.T) {
	handler := NewHandler(&fakeService{})
	router := chi.NewRouter()
	handler.RegisterRoutes(router)

	req := httptest.NewRequest(http.MethodGet, "/orgs/org-456/resources", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "user-123", "org-123", string(auth.RoleMember)))
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status=%d, want 403", w.Code)
	}
}

func TestListResourcesRejectsMissingAuthContext(t *testing.T) {
	handler := NewHandler(&fakeService{})
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-123/resources", nil)
	w := httptest.NewRecorder()

	handler.ListResources(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d, want 401", w.Code)
	}
}

func TestListResourcesMapsServiceFailure(t *testing.T) {
	handler := NewHandler(&fakeService{err: errors.New("store unavailable")})
	req := httptest.NewRequest(http.MethodGet, "/orgs/org-123/resources", nil)
	req = req.WithContext(auth.ContextWithIdentity(req.Context(), "user-123", "org-123", string(auth.RoleMember)))
	w := httptest.NewRecorder()

	handler.ListResources(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status=%d, want 500", w.Code)
	}
}

type fakeService struct {
	resources []Resource
	err       error
}

func (s *fakeService) List(ctx context.Context, orgID string) ([]Resource, error) {
	if s.err != nil {
		return nil, s.err
	}
	return s.resources, nil
}
