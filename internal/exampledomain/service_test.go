package exampledomain

import (
	"context"
	"errors"
	"testing"
)

func TestServiceListReturnsOrgScopedResources(t *testing.T) {
	store := &fakeStore{
		resources: []Resource{
			{ID: "res-1", OrgID: "org-123", Name: "First resource"},
		},
	}
	service := NewService(store)

	resources, err := service.List(context.Background(), "org-123")
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}

	if len(resources) != 1 {
		t.Fatalf("len(resources)=%d, want 1", len(resources))
	}
	if resources[0].OrgID != "org-123" {
		t.Fatalf("OrgID=%q, want org-123", resources[0].OrgID)
	}
}

func TestServiceListRejectsMissingOrgID(t *testing.T) {
	service := NewService(&fakeStore{})

	_, err := service.List(context.Background(), "")

	if !errors.Is(err, ErrMissingOrgID) {
		t.Fatalf("err=%v, want ErrMissingOrgID", err)
	}
}

type fakeStore struct {
	resources []Resource
}

func (s *fakeStore) ListByOrg(ctx context.Context, orgID string) ([]Resource, error) {
	return s.resources, nil
}
