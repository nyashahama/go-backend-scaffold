# Startup Hardening Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Harden the scaffold's runtime defaults and delivery pipeline so it is a stronger startup handoff baseline with safer proxy handling, better auth abuse resistance, tighter CORS behavior, and practical CI/release security checks.

**Architecture:** Keep the existing chi, Redis, and JWT architecture, but tighten the middleware and config boundaries where unsafe assumptions exist today. The implementation adds explicit trusted-proxy configuration, route-aware/auth-aware throttling, safer CORS header emission, and incremental GitHub Actions hardening without forcing adopters into a different auth or deployment model.

**Tech Stack:** Go, chi, Redis, pgx/sqlc, Prometheus, slog, Docker, GitHub Actions

---

### Task 1: Add Trusted Proxy Configuration And Safe Client IP Resolution

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`
- Modify: `internal/middleware/ratelimit.go`
- Modify: `internal/middleware/ratelimit_test.go`

- [ ] **Step 1: Write failing config tests for trusted proxy settings**

```go
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
```

- [ ] **Step 2: Run config tests to verify they fail**

Run: `go test ./internal/config -run 'TestLoad_(ParsesTrustedProxyCIDRs|RejectsInvalidTrustedProxyCIDR)' -v`
Expected: FAIL because the config fields and parsing do not exist yet.

- [ ] **Step 3: Write failing middleware tests for forwarded-header trust**

```go
func TestClientIP_IgnoresForwardedHeadersByDefault(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	ip := clientIP(req, ClientIPOptions{})

	if ip != "10.0.0.5" {
		t.Fatalf("clientIP=%q, want %q", ip, "10.0.0.5")
	}
}

func TestClientIP_UsesForwardedHeadersFromTrustedProxy(t *testing.T) {
	_, network, _ := net.ParseCIDR("10.0.0.0/8")
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = "10.0.0.5:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.9")

	ip := clientIP(req, ClientIPOptions{
		TrustProxyHeaders: true,
		TrustedProxies:    []*net.IPNet{network},
	})

	if ip != "203.0.113.9" {
		t.Fatalf("clientIP=%q, want %q", ip, "203.0.113.9")
	}
}
```

- [ ] **Step 4: Run middleware tests to verify they fail**

Run: `go test ./internal/middleware -run 'TestClientIP_(IgnoresForwardedHeadersByDefault|UsesForwardedHeadersFromTrustedProxy)' -v`
Expected: FAIL because `ClientIPOptions` and the new `clientIP` behavior do not exist yet.

- [ ] **Step 5: Implement config parsing and trusted client IP resolution**

```go
type Config struct {
	// existing fields...
	TrustProxyHeaders bool
	TrustedProxyCIDRs []string
}

type ClientIPOptions struct {
	TrustProxyHeaders bool
	TrustedProxies    []*net.IPNet
}
```

Implement:

- boolean parsing for `TRUST_PROXY_HEADERS`
- CIDR parsing/validation for `TRUSTED_PROXY_CIDRS`
- `clientIP(r, opts)` that only trusts forwarded headers when the direct peer IP is in the trusted proxy set

- [ ] **Step 6: Run focused config and middleware tests to verify they pass**

Run: `go test ./internal/config ./internal/middleware -run 'TestLoad_|TestClientIP_' -v`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add internal/config/config.go internal/config/config_test.go internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go
git commit -m "feat: add explicit trusted proxy handling"
```

### Task 2: Add Route-Aware And Auth-Specific Rate Limiting

**Files:**
- Modify: `internal/middleware/ratelimit.go`
- Modify: `internal/middleware/ratelimit_test.go`
- Modify: `internal/server/router.go`
- Modify: `internal/server/router_test.go`

- [ ] **Step 1: Write failing tests for auth-specific throttles**

```go
func TestRateLimit_UsesEmailScopedKeyForLogin(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", strings.NewReader(`{"email":"A.User@example.com"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.7:1234"

	key, ok := rateLimitKey(req, RateLimitPolicy{
		Scope: RateLimitScopeAuthEmailIP,
	})

	if !ok {
		t.Fatal("expected a rate limit key")
	}
	if !strings.Contains(key, "a.user@example.com") {
		t.Fatalf("key=%q, want normalized email to be included", key)
	}
}

func TestRateLimit_UsesStricterPolicyForLogin(t *testing.T) {
	policy := policyForRequest(httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil))
	if policy.Limit >= 100 {
		t.Fatalf("login limit=%d, want stricter than global limit", policy.Limit)
	}
}
```

- [ ] **Step 2: Run middleware tests to verify they fail**

Run: `go test ./internal/middleware -run 'TestRateLimit_(UsesEmailScopedKeyForLogin|UsesStricterPolicyForLogin)' -v`
Expected: FAIL because per-route policy selection and email-scoped keys do not exist yet.

- [ ] **Step 3: Implement layered rate-limit policies**

```go
type RateLimitScope string

const (
	RateLimitScopeIP          RateLimitScope = "ip"
	RateLimitScopeAuthEmailIP RateLimitScope = "auth_email_ip"
)

type RateLimitPolicy struct {
	Name   string
	Limit  int
	Window time.Duration
	Scope  RateLimitScope
}
```

Implement:

- default global policy for normal routes
- stricter policies for login, refresh, forgot-password, reset-password, and register
- email-normalized keying for JSON bodies that contain `email`
- per-request body restoration after inspection

- [ ] **Step 4: Run focused middleware and router tests**

Run: `go test ./internal/middleware ./internal/server -run 'TestRateLimit_|TestNewRouter_' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/ratelimit.go internal/middleware/ratelimit_test.go internal/server/router.go internal/server/router_test.go
git commit -m "feat: harden auth rate limiting"
```

### Task 3: Tighten CORS Header Emission

**Files:**
- Modify: `internal/middleware/cors.go`
- Add: `internal/middleware/cors_test.go`

- [ ] **Step 1: Write failing CORS tests**

```go
func TestCORS_EmitsCredentialHeadersOnlyForAllowedOrigins(t *testing.T) {
	handler := CORS([]string{"https://app.example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))

	req := httptest.NewRequest(http.MethodOptions, "/api/v1/auth/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if got := w.Header().Get("Access-Control-Allow-Credentials"); got != "" {
		t.Fatalf("allow credentials=%q, want empty", got)
	}
}
```

- [ ] **Step 2: Run CORS tests to verify they fail**

Run: `go test ./internal/middleware -run TestCORS_ -v`
Expected: FAIL because the middleware currently emits credential headers regardless of origin match.

- [ ] **Step 3: Implement safer CORS behavior**

```go
allowed := origins[origin]
if allowed {
	w.Header().Set("Access-Control-Allow-Origin", origin)
	w.Header().Set("Access-Control-Allow-Credentials", "true")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
	w.Header().Set("Access-Control-Max-Age", "86400")
}
```

Keep `Vary: Origin` and functional preflight handling.

- [ ] **Step 4: Run middleware tests**

Run: `go test ./internal/middleware -run 'TestCORS_|TestRateLimit_|TestClientIP_' -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/middleware/cors.go internal/middleware/cors_test.go
git commit -m "feat: tighten cors defaults"
```

### Task 4: Harden CI And Release Workflows

**Files:**
- Modify: `.github/workflows/ci.yml`
- Modify: `.github/workflows/release.yml`
- Modify: `Makefile`
- Modify: `docs/startup-readiness.md`
- Modify: `docs/adoption-checklist.md`
- Modify: `README.md`

- [ ] **Step 1: Write the workflow changes required by the design**

Add:

- dependency vulnerability scanning in CI
- container image scan in CI or release after build
- stronger release verification before publish
- continued use of `make ready-for-adopters` as the human-facing local gate

- [ ] **Step 2: Update maintainer and adopter docs**

Document:

- explicit proxy trust configuration
- stronger auth-throttle defaults
- security checks included in CI/release
- what adopters still must own themselves

- [ ] **Step 3: Run lightweight verification on workflow and docs changes**

Run: `bash -n scripts/ci/bootstrap-smoke.sh`
Expected: PASS

Run: `go test ./cmd/server ./internal/server ./internal/middleware ./internal/config -v`
Expected: PASS

- [ ] **Step 4: Commit**

```bash
git add .github/workflows/ci.yml .github/workflows/release.yml Makefile README.md docs/startup-readiness.md docs/adoption-checklist.md
git commit -m "ci: add security verification to startup scaffold"
```

### Task 5: Full Verification

**Files:**
- Modify: none

- [ ] **Step 1: Run the complete unit test suite**

Run: `go test ./...`
Expected: PASS

- [ ] **Step 2: Run the linter**

Run: `golangci-lint run ./...`
Expected: PASS

- [ ] **Step 3: Run readiness verification if environment allows**

Run: `make ready-for-adopters`
Expected: PASS, or report the exact external dependency limitation if the environment prevents full execution.

- [ ] **Step 4: Summarize any residual risks**

Document any remaining non-goals:

- product-specific authorization
- deployment-specific proxy/network configuration
- startup-specific compliance and legal controls
