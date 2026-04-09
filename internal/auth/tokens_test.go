package auth

import (
	"encoding/hex"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "test-secret-key-for-testing-only"
const (
	expectedAccessTokenIssuer   = "go-backend-scaffold"
	expectedAccessTokenAudience = "go-backend-scaffold-api"
)

func TestGenerateAccessToken_ValidClaims(t *testing.T) {
	tok, err := GenerateAccessToken("user-123", "org-456", "admin", 3, testSecret, 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	claims, err := ValidateAccessToken(tok, testSecret)
	if err != nil {
		t.Fatalf("expected valid token, got: %v", err)
	}
	if claims.Subject != "user-123" {
		t.Errorf("sub = %q, want %q", claims.Subject, "user-123")
	}
	if claims.OrgID != "org-456" {
		t.Errorf("org_id = %q, want %q", claims.OrgID, "org-456")
	}
	if claims.Role != "admin" {
		t.Errorf("role = %q, want %q", claims.Role, "admin")
	}
	if claims.TokenVersion != 3 {
		t.Errorf("token_version = %d, want 3", claims.TokenVersion)
	}
	if claims.Issuer != expectedAccessTokenIssuer {
		t.Errorf("issuer = %q, want %q", claims.Issuer, expectedAccessTokenIssuer)
	}
	if len(claims.Audience) != 1 || claims.Audience[0] != expectedAccessTokenAudience {
		t.Errorf("audience = %v, want [%q]", claims.Audience, expectedAccessTokenAudience)
	}
}

func TestGenerateAccessToken_EmptySecret(t *testing.T) {
	_, err := GenerateAccessToken("user-1", "org-1", "admin", 0, "", 15*time.Minute)
	if err == nil {
		t.Fatal("expected error for empty secret, got nil")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	tok, err := GenerateAccessToken("user-1", "org-1", "admin", 1, "secret-a", 15*time.Minute)
	if err != nil {
		t.Fatalf("unexpected error generating token: %v", err)
	}
	_, err = ValidateAccessToken(tok, "secret-b")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	claims := Claims{
		OrgID: "org-1",
		Role:  "admin",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to build expired token: %v", err)
	}
	_, err = ValidateAccessToken(tok, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_TamperedSignature(t *testing.T) {
	tok, _ := GenerateAccessToken("user-1", "org-1", "admin", 1, testSecret, 15*time.Minute)
	tampered := tok[:len(tok)-4] + "xxxx"
	_, err := ValidateAccessToken(tampered, testSecret)
	if err == nil {
		t.Fatal("expected error for tampered token, got nil")
	}
}

func TestValidateAccessToken_WrongAudience(t *testing.T) {
	claims := Claims{
		OrgID:        "org-1",
		Role:         "admin",
		TokenVersion: 1,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "user-1",
			Issuer:    expectedAccessTokenIssuer,
			Audience:  []string{"wrong-audience"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
		},
	}
	tok, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to build token: %v", err)
	}
	_, err = ValidateAccessToken(tok, testSecret)
	if err == nil {
		t.Fatal("expected error for wrong audience, got nil")
	}
}

func TestGenerateRefreshToken_LengthAndEntropy(t *testing.T) {
	tok, err := GenerateRefreshToken()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tok) != 64 {
		t.Errorf("token length = %d, want 64", len(tok))
	}
	if _, err := hex.DecodeString(tok); err != nil {
		t.Errorf("token is not valid hex: %v", err)
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	tok1, _ := GenerateRefreshToken()
	tok2, _ := GenerateRefreshToken()
	if tok1 == tok2 {
		t.Error("expected unique tokens, got identical values")
	}
}
