package exampledomain

import "context"

// Store defines the persistence methods this domain needs.
type Store interface {
	ListByOrg(ctx context.Context, orgID string) ([]Resource, error)
}
