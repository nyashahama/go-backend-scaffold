package auth

import (
	"time"
)

// ContextKey is the type for auth-related context keys.
type ContextKey string

const (
	UserIDKey ContextKey = "user_id"
	OrgIDKey  ContextKey = "org_id"
	RoleKey   ContextKey = "role"
)

// Claims holds JWT payload fields.
type Claims struct {
	OrgID   string
	Role    string
	Subject string
}

// ValidateAccessToken placeholder — implemented in Task 10.
func ValidateAccessToken(tokenStr, secret string) (*Claims, error) {
	panic("not implemented — replaced by Task 10")
}

// GenerateAccessToken placeholder — implemented in Task 10.
func GenerateAccessToken(userID, orgID, role, secret string, expiry time.Duration) (string, error) {
	panic("not implemented — replaced by Task 10")
}

// GenerateRefreshToken placeholder — implemented in Task 10.
func GenerateRefreshToken() (string, error) {
	panic("not implemented — replaced by Task 10")
}
