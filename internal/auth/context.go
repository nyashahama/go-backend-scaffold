package auth

import (
	"context"
	"net/http"
)

// contextKey is an unexported type for context keys in this package.
// Using a struct prevents collisions with keys from other packages.
type contextKey struct{ name string }

var (
	userIDKey = &contextKey{"user_id"}
	orgIDKey  = &contextKey{"org_id"}
	roleKey   = &contextKey{"role"}
)

// Identity holds the caller's identity extracted from a validated JWT.
type Identity struct {
	UserID string
	OrgID  string
	Role   string
}

// ContextWithIdentity stores user_id, org_id, and role in the context.
func ContextWithIdentity(ctx context.Context, userID, orgID, role string) context.Context {
	ctx = context.WithValue(ctx, userIDKey, userID)
	ctx = context.WithValue(ctx, orgIDKey, orgID)
	ctx = context.WithValue(ctx, roleKey, role)
	return ctx
}

// ContextWithClaims stores JWT claims fields in the context.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	if claims == nil {
		return ctx
	}
	return ContextWithIdentity(ctx, claims.Subject, claims.OrgID, claims.Role)
}

// IdentityFromContext reads the identity stored by ContextWithIdentity.
func IdentityFromContext(ctx context.Context) (Identity, bool) {
	identity := Identity{
		UserID: valueFromContext(ctx, userIDKey),
		OrgID:  valueFromContext(ctx, orgIDKey),
		Role:   valueFromContext(ctx, roleKey),
	}
	if identity.UserID == "" || identity.OrgID == "" || identity.Role == "" {
		return Identity{}, false
	}
	return identity, true
}

// IdentityFromRequest is a convenience wrapper over IdentityFromContext.
func IdentityFromRequest(r *http.Request) (Identity, bool) {
	if r == nil {
		return Identity{}, false
	}
	return IdentityFromContext(r.Context())
}

func valueFromContext(ctx context.Context, key *contextKey) string {
	value, _ := ctx.Value(key).(string)
	return value
}
