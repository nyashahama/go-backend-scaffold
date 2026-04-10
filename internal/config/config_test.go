package config

import "testing"

func TestLoad_RejectsPlaceholderJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/scaffold?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "changeme-use-openssl-rand-base64-32")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for placeholder JWT secret, got nil")
	}
}

func TestLoad_RejectsShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/scaffold?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "short-secret")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for short JWT secret, got nil")
	}
}

func TestLoad_LoadsResendEmailConfig(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/scaffold?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "test-secret-that-is-long-enough-for-config")
	t.Setenv("RESEND_API_KEY", "re_test_123")
	t.Setenv("EMAIL_FROM", "no-reply@example.com")
	t.Setenv("EMAIL_FROM_NAME", "Example App")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.ResendAPIKey != "re_test_123" {
		t.Fatalf("ResendAPIKey=%q, want re_test_123", cfg.ResendAPIKey)
	}
	if cfg.EmailFrom != "no-reply@example.com" {
		t.Fatalf("EmailFrom=%q, want no-reply@example.com", cfg.EmailFrom)
	}
	if cfg.EmailFromName != "Example App" {
		t.Fatalf("EmailFromName=%q, want Example App", cfg.EmailFromName)
	}
}

func TestLoad_ParsesTrustedProxyCIDRs(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/scaffold?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "test-secret-that-is-long-enough-for-config")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("TRUSTED_PROXY_CIDRS", "10.0.0.0/8,192.168.0.0/16")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cfg.TrustProxyHeaders {
		t.Fatal("expected TrustProxyHeaders to be true")
	}
	if len(cfg.TrustedProxyCIDRs) != 2 {
		t.Fatalf("len(TrustedProxyCIDRs)=%d, want 2", len(cfg.TrustedProxyCIDRs))
	}
}

func TestLoad_RejectsInvalidTrustedProxyCIDR(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://user:password@localhost:5432/scaffold?sslmode=disable")
	t.Setenv("REDIS_URL", "redis://localhost:6379")
	t.Setenv("JWT_SECRET", "test-secret-that-is-long-enough-for-config")
	t.Setenv("TRUST_PROXY_HEADERS", "true")
	t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid trusted proxy cidr")
	}
}
