package exampledomain

import (
	"context"
	"errors"
)

var ErrMissingOrgID = errors.New("exampledomain: missing org id")

type Servicer interface {
	List(ctx context.Context, orgID string) ([]Resource, error)
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) List(ctx context.Context, orgID string) ([]Resource, error) {
	if orgID == "" {
		return nil, ErrMissingOrgID
	}
	return s.store.ListByOrg(ctx, orgID)
}
