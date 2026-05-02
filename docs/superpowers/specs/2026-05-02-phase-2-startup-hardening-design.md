# Phase 2 Startup Hardening Design

## Goal

Make the scaffold safer to adopt in a startup environment by removing implicit trust assumptions and making ownership-sensitive identity values first-class configurable, while preserving the small scope and low coupling expected from a startup starter template.

## Scope

This phase includes:

1. Runtime middleware hardening for request identity and abuse protection.
2. JWT token identity metadata configuration (issuer/audience) and startup initialization support.
3. Safer release and CI validation for vulnerabilities and tokenized build outputs.

This phase does not add new domain features, change auth model, or add a full security compliance framework.

## Current Blockers

1. Reverse-proxy client identity is inferred from forwarded headers in a narrowly scoped but still implicit way (`X-Forwarded-For` only when remote address is loopback/private).
2. JWT issuer/audience are hardcoded in `internal/auth/tokens.go`, which makes token namespace identity coupled to scaffold naming.
3. Rate limiting applies one global policy and cannot be tuned for sensitive authentication endpoints.
4. `CORS` always emits credential headers and method/header policy even for disallowed origins.
5. `scripts/init-template.sh` rewrites module import paths only; scaffold ownership labels and security identity are still manual.
6. CI/release checks include lint and tests but no dependency/container risk checks.

## Target Design

### 1) Make proxy trust explicit in config

`internal/config/config.go` will gain:

- `TrustProxy` (`TRUST_PROXY`) default `false`.
- Optional `TrustedProxies` (`TRUSTED_PROXY_CIDRS`) as optional CIDR allowlist.

Runtime behavior:

- `RateLimit` will never use `X-Forwarded-For`/`X-Real-IP` unless `TrustProxy` is true and the direct peer is either loopback/private **or** within `TrustedProxies` when provided.
- If neither condition is met, client identity is derived from `RemoteAddr` only.
- Invalid CIDR values fail config loading with an explicit error.

### 2) Add rate-limit tiers and auth-route sensitivity

`internal/middleware/ratelimit.go` will gain route-level policy selection:

- Keep global middleware with broad limits for non-auth routes (existing behavior preserved).
- Add a stricter auth-policy middleware attached inside `internal/server/router.go` under `/api/v1/auth` for:
  - `POST /auth/login`
  - `POST /auth/refresh`
  - `POST /auth/forgot-password`
  - `POST /auth/reset-password`
- The stricter limiter keys requests by both identity inputs when possible:
  - if request body parses `email`, include normalized email in key prefix
  - fallback to IP key if body is not parseable.

Defaults in this phase:

- auth throttle: 10 req/minute with fail-closed on Redis error
- global fallback: retain current limit of 100 req/minute

### 3) Tighten CORS header behavior

`internal/middleware/cors.go` will only emit credentials-related headers for matching allowed origins:

- `Access-Control-Allow-Credentials` set only when origin is in allowlist.
- `Access-Control-Allow-Headers` and methods still set only for matched allowed origin.
- For non-matching origins, keep `Vary: Origin` and avoid signaling permission to send credentials.

### 4) Configure app identity and JWT namespace

`internal/auth/tokens.go`, `internal/auth/service.go`, and `internal/middleware/auth.go` will be updated to use config-driven token names:

- `JWT_ACCESS_TOKEN_ISSUER` default `go-backend-scaffold`
- `JWT_ACCESS_TOKEN_AUDIENCE` default `go-backend-scaffold-api`
- values carried in `auth.Service` and validation middleware to keep generate/validate checks aligned.

`scripts/init-template.sh` will also rewrite these values from placeholder fields when present, so new adopters can initialize in one pass rather than hand-editing many references.

### 5) Improve CI risk checks without changing the product surface

`.github/workflows/ci.yml` and `.github/workflows/release.yml` will gain lightweight but practical checks:

- dependency vulnerability checks for Go module graph in CI
- image scan for the scaffold build artifact in CI
- explicit dependency on the documented local readiness gate (`make ready-for-adopters`) before release publish path.

## Functional Requirements

1. A direct server process must remain correctly rate-limited when `TRUST_PROXY=false`.
2. A server behind a proxy can enable trust explicitly and still pass security-sensitive tests.
3. Public auth endpoints must be materially harder to brute-force than general API endpoints.
4. `ValidateAccessToken` must fail when issuer/audience do not match configured values.
5. CORS responses for invalid origin must not advertise credential allowance.
6. `scripts/init-template.sh` leaves no unresolved scaffold-specific auth token identity fields after initialization.
7. CI must surface scanning failures as release-blocking failures.

## Non-Functional Requirements

- Changes should keep file count low and preserve existing architecture.
- Tests should remain unit-tested with focused middleware tests and integration assertions for auth-path protections.
- No behavior changes for health and readiness endpoints beyond improved security semantics.

## Success Criteria

The phase is complete when all of the following pass:

1. `go test ./... -race` passes.
2. `make ready-for-adopters` passes.
3. Middleware tests explicitly cover:
   - forwarded header ignored by default,
   - forwarded header used only under explicit trust,
   - CORS credential behavior for matching and non-matching origins,
   - auth-route limiter applies stricter windows.
4. `scripts/init-template.sh` updates both Go module references and identity placeholders.
5. CI executes the added scans and blocks on failures.
