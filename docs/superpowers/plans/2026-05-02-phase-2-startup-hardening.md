# Phase 2 Startup Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the scaffold safer and easier to hand off by making trust, rate-limits, and token identity explicit, plus adding adoption-ready security checks.

**Architecture:** Keep the existing package layout and middleware chain. Add minimal, composable middleware behavior behind existing interfaces and config.

**Tech Stack:** Go standard library, chi, redis, jwt-go, GitHub Actions.

---

## Task 1: Add explicit proxy trust settings to config

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Add: `TRUST_PROXY` and `TRUSTED_PROXY_CIDRS` values in `tests` setup

- [ ] Step 1: Add config fields and defaults

```go
type Config struct {
    // ...
    TrustProxy      bool
    TrustedProxyCIDRs []string
}

trustProxy := os.Getenv("TRUST_PROXY")
cfg.TrustProxy = strings.EqualFold(trustProxy, "true")
trusted := os.Getenv("TRUSTED_PROXY_CIDRS")
if trusted != "" {
    for _, cidr := range strings.Split(trusted, ",") {
        cidr = strings.TrimSpace(cidr)
        if _, _, err := net.ParseCIDR(cidr); err != nil {
            return nil, fmt.Errorf("invalid TRUSTED_PROXY_CIDRS value %q: %w", cidr, err)
        }
        cfg.TrustedProxyCIDRs = append(cfg.TrustedProxyCIDRs, cidr)
    }
}
```

- [ ] Step 2: Add negative config tests

```go
func TestLoad_RejectsInvalidTrustedProxyCIDR(t *testing.T) {
    t.Setenv("DATABASE_URL", "postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable")
    t.Setenv("REDIS_URL", "redis://localhost:6379")
    t.Setenv("JWT_SECRET", "test-secret-that-is-long-enough-for-config")
    t.Setenv("TRUSTED_PROXY_CIDRS", "not-a-cidr")

    _, err := Load()
    if err == nil {
        t.Fatal("expected invalid CIDR to fail")
    }
}
```

- [ ] Step 3: Run tests

Run: `go test ./internal/config -v`

- [ ] Step 4: Commit

```bash
git add internal/config/config.go internal/config/config_test.go
git commit -m "feat(config): add explicit trusted proxy config and validation"
```

## Task 2: Make rate limiting trust headers only when configured

**Files:**
- Modify: `internal/middleware/ratelimit.go`
- Modify: `internal/middleware/ratelimit_test.go`
- Modify: `internal/server/router.go` (for wiring)

- [ ] Step 1: Expand client identity resolution with proxy trust guardrails

```go
func clientIP(r *http.Request, trustProxy bool, trustedProxyCIDRs []string) string {
    host, _, err := net.SplitHostPort(r.RemoteAddr)
    if err != nil { host = r.RemoteAddr }
    ip := net.ParseIP(host)
    if ip == nil {
        return host
    }
    if !trustProxy {
        return host
    }
    if !isAllowedProxy(ip, trustedProxyCIDRs) && !(ip.IsLoopback() || ip.IsPrivate()) {
        return host
    }
    if forwarded := firstForwardedIP(r.Header.Get("X-Forwarded-For")); forwarded != "" {
        return forwarded
    }
    if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" && net.ParseIP(realIP) != nil {
        return realIP
    }
    return host
}
```

- [ ] Step 2: Wire trust flags into `RateLimit`

```go
func RateLimit(rdb *redis.Client, limit int, window time.Duration, trustProxy bool, trustedProxyCIDRs []string) func(http.Handler) http.Handler
```

- [ ] Step 3: Add tests for default safety and explicit-trust behavior

```go
func TestRateLimit_UsesForwardedHeadersOnlyWhenTrusted(t *testing.T) { ... }
func TestRateLimit_DropsForwardedHeadersWhenNotTrusted(t *testing.T) { ... }
```

- [ ] Step 4: Update middleware callsites and run targeted tests

Update `internal/server/router.go`:

```go
r.Use(middleware.RateLimit(rdb, 100, time.Minute, cfg.TrustProxy, cfg.TrustedProxyCIDRs))
```

Run: `go test ./internal/middleware ./internal/server -v`

- [ ] Step 5: Commit

```bash
git add internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go internal/server/router.go
git commit -m "feat(middleware): make forwarded-ip usage explicit and configurable"
```

## Task 3: Add auth-route-sensitive rate limiting

**Files:**
- Modify: `internal/server/router.go`
- Modify: `internal/middleware/ratelimit.go`
- Modify: `internal/middleware/ratelimit_test.go`

- [ ] Step 1: Add helper middleware that accepts optional body-based key hint

```go
func RateLimitByEmail(rdb *redis.Client, limit int, window time.Duration, trustProxy bool, trustedProxyCIDRs []string) func(http.Handler) http.Handler
```

- [ ] Step 2: Add request body parser for auth routes

Use `io.ReadAll` with `io.NopCloser` restore when path is sensitive auth route and key by `email` field if parseable.

- [ ] Step 3: Apply on `/api/v1/auth` routes in router

```go
r.Route("/auth", func(r chi.Router) {
    r.Use(middleware.RateLimitByEmail(rdb, 10, time.Minute, cfg.TrustProxy, cfg.TrustedProxyCIDRs))
    h.Auth.RegisterRoutes(r, middleware.Auth(cfg.JWTSecret, users))
})
```

- [ ] Step 4: Add route-protection regression test

```go
func TestAuthRouteUsesStricterRateLimit(t *testing.T) { ... }
```

- [ ] Step 5: Run tests and commit

Run: `go test ./internal/server ./internal/middleware -v`

```bash
git add internal/server/router.go internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go
git commit -m "feat(middleware): add stricter auth-route limiting"
```

## Task 4: Make CORS headers conditional on allowed origin

**Files:**
- Modify: `internal/middleware/cors.go`
- Modify: `internal/middleware/ratelimit_test.go` (if you keep unit test helper shared)

- [ ] Step 1: Emit credentials/header allowances only when origin is allowed

```go
if origins[origin] {
    w.Header().Set("Access-Control-Allow-Origin", origin)
    w.Header().Set("Access-Control-Allow-Credentials", "true")
    w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
    w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
    w.Header().Set("Access-Control-Max-Age", "86400")
}
```

- [ ] Step 2: Add tests

```go
func TestCORS_AllowsCredentialsOnlyForAllowedOrigin(t *testing.T) { ... }
```

- [ ] Step 3: Run and commit

Run: `go test ./internal/middleware -v`

```bash
git add internal/middleware/cors.go internal/middleware/cors_test.go
git commit -m "fix(cors): only emit credential headers for allowed origins"
```

## Task 5: Make JWT issuer/audience configurable and template-safe

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/auth/tokens.go`
- Modify: `internal/auth/service.go`
- Modify: `internal/middleware/auth.go`
- Modify: `internal/auth/tokens_test.go`
- Modify: `internal/middleware/auth_test.go` (if present, add/extend token validation coverage as needed)
- Modify: `scripts/init-template.sh`
- Modify: `docs/startup-readiness.md`, `docs/adoption-checklist.md`, `README.md`

- [ ] Step 1: Add config fields with defaults

```go
JWTAccessTokenIssuer: getEnv("JWT_ACCESS_TOKEN_ISSUER", "go-backend-scaffold"),
JWTAccessTokenAudience: getEnv("JWT_ACCESS_TOKEN_AUDIENCE", "go-backend-scaffold-api"),
```

- [ ] Step 2: Thread values through auth generation/validation

```go
func GenerateAccessToken(userID, orgID, role string, tokenVersion int32, secret, issuer, audience string, expiry time.Duration) (string, error)
func ValidateAccessToken(tokenStr, secret, issuer, audience string) (*Claims, error)
```

- [ ] Step 3: Update service/auth middleware callsites

Pass config values from `cfg` into service constructor and auth middleware usage.

- [ ] Step 4: Add tests for configured mismatch behavior

```go
func TestValidateAccessToken_RejectsWrongIssuerOrAudience(t *testing.T) { ... }
```

- [ ] Step 5: Extend `scripts/init-template.sh` replacements

Add replacements for placeholders:

- `go-backend-scaffold`
- `go-backend-scaffold-api`
- docker image and registry placeholders where present

- [ ] Step 6: Run tests and commit

Run: `go test ./internal/auth ./internal/middleware -v`

```bash
git add internal/config/config.go internal/auth/tokens.go internal/auth/service.go internal/middleware/auth.go internal/auth/tokens_test.go scripts/init-template.sh
git commit -m "feat(auth): make token issuer/audience configurable and template-ready"
```

## Task 6: Add release/security scanning to CI and release gate

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`

- [ ] Step 1: Add vulnerability scan in CI

```yaml
      - name: Scan dependencies
        run: go run golang.org/x/vuln/cmd/govulncheck@latest ./...
```

- [ ] Step 2: Add image scanning step

Build to a local tag, run scanner, fail on issues.

- [ ] Step 3: Gate release on local readiness

Run readiness command before publish/push step:

```yaml
run: make ready-for-adopters
```

- [ ] Step 4: Run and commit

Run: `go test ./... -race` and verify CI workflow YAML with `make ready-for-adopters` expectations via local review.

```bash
git add .github/workflows/ci.yml .github/workflows/release.yml
git commit -m "ci: add dependency and container scanning and readiness gate checks"
```

## Self-review

- [ ] Confirm every spec section maps to a task above.
- [ ] Scan for placeholders or vague placeholders in test names or commands.
- [ ] Verify file paths match actual project layout.
- [ ] Verify each commit message maps to a single cohesive behavior area.
- [ ] No task assumes a code-level dependency not added in previous tasks.

Plan complete and saved to `docs/superpowers/plans/2026-05-02-phase-2-startup-hardening.md`.
