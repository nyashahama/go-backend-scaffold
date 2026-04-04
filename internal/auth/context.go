package auth

import (
	"context"
	"net/http"
)

// Identity holds the caller's identity extracted from a validated JWT.
type Identity struct {
	UserID string
	OrgID  string
	Role   string
}

func ContextWithIdentity(ctx context.Context, userID, orgID, role string) context.Context {
	panic("not implemented — replaced by Task 11")
}

func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	panic("not implemented — replaced by Task 11")
}

func IdentityFromContext(ctx context.Context) (Identity, bool) {
	panic("not implemented — replaced by Task 11")
}

func IdentityFromRequest(r *http.Request) (Identity, bool) {
	panic("not implemented — replaced by Task 11")
}
