# Startup Readiness Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make this scaffold safe and predictable enough that an arbitrary startup team can clone it, boot it, verify it, and begin product work without first reverse-engineering the template.

**Architecture:** Treat readiness as a product problem, not just a code problem. The work is split into deterministic bootstrap, real router-level smoke coverage, CI quality gates, and adopter-facing template ergonomics so the safest path is also the easiest path.

**Tech Stack:** Go, chi, pgx/sqlc, Redis, Postgres, Docker Compose, GitHub Actions, goose, slog

---

## File Map

- Create: `.github/workflows/ci.yml`
  Runs unit tests, integration tests, bootstrap smoke checks, and Docker build verification on every push/PR.
- Create: `scripts/ci/bootstrap-smoke.sh`
  Reproduces the clean-clone developer path end-to-end with isolated Compose resources and deterministic service readiness.
- Create: `scripts/init-template.sh`
  Rewrites module path and project naming safely so adopters do not hand-edit imports.
- Create: `tests/integration/router_smoke_test.go`
  Exercises the real router for the minimal “startup can build on this” path.
- Create: `docs/adoption-checklist.md`
  Explicit “before first production deploy” checklist for adopters.
- Modify: `Makefile`
  Add stable readiness targets such as `smoke`, `test-ci`, and `bootstrap-smoke`.
- Modify: `README.md`
  Replace aspirational language with a verified bootstrap path and explicit operating assumptions.
- Modify: `.env.example`
  Keep it aligned with `docker-compose.yml`, runtime validation, and email/metrics options.
- Modify: `docker-compose.yml`
  Ensure service credentials and ports match the documented local path while allowing smoke-test isolation.
- Modify: `tests/integration/testhelpers_test.go`
  Reduce environment flakiness and centralize setup/teardown expectations.
- Modify: `internal/server/router_test.go`
  Add coverage for infra endpoints and minimal router wiring assumptions.
- Modify: `cmd/server/main_test.go`
  Add runtime configuration coverage for “random startup” production misconfiguration failures.

### Task 1: Make Bootstrap Deterministic From A Clean Clone

**Files:**
- Create: `scripts/ci/bootstrap-smoke.sh`
- Modify: `Makefile`
- Modify: `README.md`
- Modify: `.env.example`
- Modify: `docker-compose.yml`

- [ ] **Step 1: Write the failing bootstrap smoke script**

```bash
#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

SMOKE_ENV_FILE=".env.bootstrap"
cp .env.example "$SMOKE_ENV_FILE"

export COMPOSE_PROJECT_NAME="go-backend-scaffold-smoke"
export POSTGRES_PORT="${POSTGRES_PORT:-15432}"
export REDIS_PORT="${REDIS_PORT:-16379}"
export DATABASE_URL="postgres://user:change-me-local-dev@localhost:${POSTGRES_PORT}/scaffold?sslmode=disable"
export REDIS_URL="redis://localhost:${REDIS_PORT}"
export JWT_SECRET="bootstrap-smoke-secret-that-is-at-least-32-chars"

cleanup() {
  docker compose --env-file "$SMOKE_ENV_FILE" down -v
  rm -f "$SMOKE_ENV_FILE"
}
trap cleanup EXIT

docker compose --env-file "$SMOKE_ENV_FILE" up -d --wait postgres redis
make migrate-up
go test ./...
go test ./tests/integration/... -tags=integration
```

- [ ] **Step 2: Run the smoke script to verify the current bootstrap path fails in a fresh-path simulation**

Run: `bash scripts/ci/bootstrap-smoke.sh`
Expected: FAIL before all checks complete, proving the clean-clone path is not yet deterministic.

- [ ] **Step 3: Add explicit readiness-oriented Make targets**

```make
smoke:
	go test ./cmd/server ./internal/server ./internal/auth -v

bootstrap-smoke:
	bash scripts/ci/bootstrap-smoke.sh

test-ci: test
	go test ./tests/integration/... -tags=integration
```

- [ ] **Step 4: Align the documented environment and Compose defaults**

```dotenv
DATABASE_URL=postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable
REDIS_URL=redis://localhost:6379
JWT_SECRET=changeme-use-openssl-rand-base64-32
APP_BASE_URL=http://localhost:3000
ALLOWED_ORIGINS=http://localhost:3000
```

```yaml
postgres:
  environment:
    POSTGRES_USER: user
    POSTGRES_PASSWORD: change-me-local-dev
    POSTGRES_DB: scaffold
  ports:
    - "127.0.0.1:${POSTGRES_PORT:-5432}:5432"
redis:
  ports:
    - "127.0.0.1:${REDIS_PORT:-6379}:6379"
```

- [ ] **Step 5: Update the README quickstart to match the verified path exactly**

```md
### Verified Local Bootstrap

```bash
cp .env.example .env
# Replace JWT_SECRET before running the server
make docker-up
make migrate-up
make test
make test-integration
make run
```

If you want to verify the template from a clean path, run:

```bash
make bootstrap-smoke
```
```

- [ ] **Step 6: Run the smoke script again to verify bootstrap passes**

Run: `bash scripts/ci/bootstrap-smoke.sh`
Expected: PASS with unit tests and integration tests both completing successfully.

- [ ] **Step 7: Commit**

```bash
git add scripts/ci/bootstrap-smoke.sh Makefile README.md .env.example docker-compose.yml docs/superpowers/plans/2026-04-09-startup-readiness.md
git commit -m "feat: make scaffold bootstrap deterministic"
```

### Task 2: Add A Real Router-Level Startup Smoke Test

**Files:**
- Create: `tests/integration/router_smoke_test.go`
- Modify: `tests/integration/testhelpers_test.go`
- Modify: `internal/server/router_test.go`

- [ ] **Step 1: Write the failing router smoke test**

```go
func TestRouter_StartupSmokeFlow(t *testing.T) {
	prepareIntegrationState(t)
	router := newAuthRouter(t, nil)

	reg := registerViaRouter(t, router, uniqueEmail(t), testPassword)

	meReq := authRequest(http.MethodGet, "/api/v1/auth/me", reg.AccessToken, nil)
	meRes := httptest.NewRecorder()
	router.ServeHTTP(meRes, meReq)
	if meRes.Code != http.StatusOK {
		t.Fatalf("me status=%d body=%s", meRes.Code, meRes.Body.String())
	}

	refreshRes := httptest.NewRecorder()
	router.ServeHTTP(refreshRes, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", jsonReader(t, map[string]string{
		"refresh_token": reg.RefreshToken,
	})))
	if refreshRes.Code != http.StatusOK {
		t.Fatalf("refresh status=%d body=%s", refreshRes.Code, refreshRes.Body.String())
	}
}
```

- [ ] **Step 2: Run the targeted integration test and verify it fails for the missing router smoke path**

Run: `go test ./tests/integration/... -tags=integration -run TestRouter_StartupSmokeFlow -v`
Expected: FAIL because the smoke test does not exist yet.

- [ ] **Step 3: Centralize test setup so every router smoke test starts from known state**

```go
func prepareIntegrationState(t *testing.T) {
	t.Helper()
	resetAuthState(t)
	if err := testRedis.FlushDB(context.Background()).Err(); err != nil {
		t.Fatalf("flush redis: %v", err)
	}
}
```

- [ ] **Step 4: Add router tests for infra assumptions that adopters rely on**

```go
func TestNewRouter_ExposesHealthzWithoutAuth(t *testing.T) {
	router := NewRouter(cfg, logger, nil, nil, handlers)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}
}
```

- [ ] **Step 5: Run the router-level smoke suite**

Run: `go test ./internal/server ./tests/integration/... -tags=integration -run 'TestNewRouter_|TestRouter_StartupSmokeFlow' -v`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add tests/integration/router_smoke_test.go tests/integration/testhelpers_test.go internal/server/router_test.go
git commit -m "test: add startup smoke coverage"
```

### Task 3: Add CI That Verifies What README Promises

**Files:**
- Create: `.github/workflows/ci.yml`
- Modify: `Makefile`
- Modify: `README.md`

- [ ] **Step 1: Write the failing CI workflow**

```yaml
name: ci

on:
  pull_request:
  push:
    branches: [main]

jobs:
  verify:
    runs-on: ubuntu-latest
    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_USER: user
          POSTGRES_PASSWORD: change-me-local-dev
          POSTGRES_DB: scaffold
        ports: ["5432:5432"]
      redis:
        image: redis:7-alpine
        ports: ["6379:6379"]
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version-file: go.mod
      - run: cp .env.example .env
      - run: make test-ci
```

- [ ] **Step 2: Run the local equivalent to verify the gate is currently incomplete**

Run: `make test-ci`
Expected: FAIL before the full readiness target set exists.

- [ ] **Step 3: Expand the workflow to verify migrations and Docker build explicitly**

```yaml
      - name: Apply migrations
        run: make migrate-up
        env:
          DATABASE_URL: postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable

      - name: Unit and integration tests
        run: make test-ci
        env:
          DATABASE_URL: postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable
          REDIS_URL: redis://localhost:6379
          JWT_SECRET: ci-secret-that-is-at-least-32-chars

      - name: Docker build
        run: docker build -t go-backend-scaffold:ci .
```

- [ ] **Step 4: Add a badge/reference in the README only after the workflow exists**

```md
## Quality Gates

- CI verifies unit tests
- CI verifies integration tests against Postgres and Redis
- CI verifies the Docker image still builds
```

- [ ] **Step 5: Run the local equivalent again**

Run: `make test-ci && docker build -t go-backend-scaffold:ci .`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add .github/workflows/ci.yml Makefile README.md
git commit -m "ci: verify the documented startup path"
```

### Task 4: Make Template Adoption Easier Than Manual Repo Surgery

**Files:**
- Create: `scripts/init-template.sh`
- Create: `docs/adoption-checklist.md`
- Modify: `README.md`

- [ ] **Step 1: Write the failing module-init script**

```bash
#!/usr/bin/env bash
set -euo pipefail

if [ "${#}" -ne 1 ]; then
  echo "usage: scripts/init-template.sh github.com/yourorg/yourapp" >&2
  exit 1
fi

NEW_MODULE="$1"
OLD_MODULE="github.com/nyashahama/go-backend-scaffold"

go mod edit -module "$NEW_MODULE"
rg -l "$OLD_MODULE" . | xargs sed -i "s|$OLD_MODULE|$NEW_MODULE|g"
go mod tidy
```

- [ ] **Step 2: Run the script in a disposable copy and verify it currently is missing**

Run: `bash scripts/init-template.sh github.com/example/acme-api`
Expected: FAIL with “no such file” before implementation.

- [ ] **Step 3: Document the adopter contract explicitly**

```md
# Adoption Checklist

- Change the Go module path
- Replace local secrets in `.env`
- Choose an email sender before production
- Run `make bootstrap-smoke`
- Run `make test-ci`
- Confirm `/healthz`, `/readyz`, and `/metrics` behavior for your environment
```

- [ ] **Step 4: Replace the README rename step with the script**

```md
### 1. Clone and initialize

```bash
git clone https://github.com/nyashahama/go-backend-scaffold.git my-api
cd my-api
bash scripts/init-template.sh github.com/yourname/my-api
```
```

- [ ] **Step 5: Verify the init script behavior**

Run: `bash scripts/init-template.sh github.com/example/acme-api`
Expected: PASS with `go.mod` updated and imports rewritten.

- [ ] **Step 6: Commit**

```bash
git add scripts/init-template.sh docs/adoption-checklist.md README.md
git commit -m "docs: improve template adoption flow"
```

### Task 5: Add A Final “Safe For Random Startup” Release Gate

**Files:**
- Modify: `cmd/server/main_test.go`
- Modify: `README.md`
- Modify: `Makefile`
- Create: `docs/startup-readiness.md`

- [ ] **Step 1: Write the failing runtime-safety tests**

```go
func TestValidateRuntimeConfig_ProductionRequiresAppBaseURL(t *testing.T) {
	cfg := &config.Config{Env: "production"}
	err := validateRuntimeConfig(cfg, &notification.NoopSender{})
	if err == nil {
		t.Fatal("expected production config validation error")
	}
}
```

- [ ] **Step 2: Run the targeted server tests and verify the new release gate does not exist yet**

Run: `go test ./cmd/server -run TestValidateRuntimeConfig_ProductionRequiresAppBaseURL -v`
Expected: FAIL before the full readiness gate is documented and exposed.

- [ ] **Step 3: Add a single readiness target that adopters and maintainers can trust**

```make
ready-for-adopters:
	go test ./...
	go vet ./...
	go test ./tests/integration/... -tags=integration
	docker build -t go-backend-scaffold:ready .
```

- [ ] **Step 4: Document what “ready” means in-repo**

```md
# Startup Readiness

This scaffold is ready for external adoption only when all of the following are true:
- `make ready-for-adopters` passes
- the bootstrap smoke script passes from a clean clone
- the CI workflow is green
- the adoption checklist has been completed for the target startup
```

- [ ] **Step 5: Run the final readiness target**

Run: `make ready-for-adopters`
Expected: PASS

- [ ] **Step 6: Commit**

```bash
git add cmd/server/main_test.go Makefile README.md docs/startup-readiness.md
git commit -m "docs: add startup readiness gate"
```
