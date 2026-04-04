// Package auth provides JWT token creation, validation, and context utilities.
// This stub will be replaced by the full implementation in Tasks 10–14.
package auth

import (
	"context"
	"errors"
)

// Claims holds the parsed JWT claims for an authenticated user.
type Claims struct {
	UserID string
	Email  string
	Role   string
}

type contextKey string

const claimsKey contextKey = "claims"

// ValidateAccessToken parses and validates a signed JWT access token.
// Full implementation is provided in Task 10.
func ValidateAccessToken(tokenStr, secret string) (*Claims, error) {
	return nil, errors.New("not implemented")
}

// ContextWithClaims returns a new context with the given claims attached.
func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, claimsKey, claims)
}

// ClaimsFromContext retrieves Claims from the context, if present.
func ClaimsFromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(claimsKey).(*Claims)
	return c, ok
}
