package auth

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Claims holds JWT payload fields beyond the registered set.
type Claims struct {
	OrgID        string `json:"org_id"`
	Role         string `json:"role"`
	TokenVersion int32  `json:"token_version"`
	jwt.RegisteredClaims
}

// GenerateAccessToken creates a signed HS256 JWT for the given user.
func GenerateAccessToken(userID, orgID, role string, tokenVersion int32, secret, issuer, audience string, expiry time.Duration) (string, error) {
	if secret == "" {
		return "", errors.New("auth: secret must not be empty")
	}
	if issuer == "" {
		return "", errors.New("auth: issuer must not be empty")
	}
	if audience == "" {
		return "", errors.New("auth: audience must not be empty")
	}

	jti, err := GenerateRefreshToken()
	if err != nil {
		return "", err
	}

	now := time.Now()
	claims := Claims{
		OrgID:        orgID,
		Role:         role,
		TokenVersion: tokenVersion,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   userID,
			Issuer:    issuer,
			Audience:  []string{audience},
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(expiry)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// GenerateRefreshToken returns a 32-byte cryptographically random hex string.
func GenerateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// ValidateAccessToken parses and validates a JWT, returning its claims.
func ValidateAccessToken(tokenStr, secret, issuer, audience string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(
		tokenStr,
		&Claims{},
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("unexpected signing method")
			}
			return []byte(secret), nil
		},
		jwt.WithIssuer(issuer),
		jwt.WithAudience(audience),
	)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}
