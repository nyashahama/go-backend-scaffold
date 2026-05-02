# go-backend-scaffold

A production-ready Go backend scaffold. Clone it, initialize the module path safely, and build your next API.

**Stack:** chi · pgx/v5 · sqlc · goose · JWT · Redis · Prometheus · slog · Docker · GitHub Actions

## Quickstart

### 1. Clone and initialize

```bash
git clone https://github.com/nyashahama/go-backend-scaffold.git my-api
cd my-api
# Run this once before making project-specific edits.
bash scripts/init-template.sh github.com/yourname/my-api
make check-adoption
```

### 2. Install tools

```bash
make install-tools
```

### Verified Local Bootstrap

```bash
cp .env.example .env
# Replace JWT_SECRET before running the server
make docker-up
make migrate-up
make test-ci
make run
```

If you want to verify the template from a clean path, run:

```bash
make bootstrap-smoke
make init-template-smoke
```

When you are deciding whether this scaffold is ready to hand to a random startup adopter, run:

```bash
make ready-for-adopters
```

This assumes the local toolchain is current enough to lint and test the module, typically after `make install-tools`.

Before a real deployment, complete the [adoption checklist](docs/adoption-checklist.md).
Read the explicit release gate definition in [docs/startup-readiness.md](docs/startup-readiness.md).

The server starts on `http://localhost:8080`.

For a full containerized stack, including the backend container, run:

```bash
docker compose --profile full up --build
```

## Quality Gates

GitHub Actions verifies core quality gates for this scaffold:

- database migrations apply cleanly against a fresh Postgres service
- lint passes, and `make test-ci` passes, which runs the repository test sweep plus integration tests with race detection
- `govulncheck ./...` runs against reachable Go code
- `make docker-build IMAGE_TAG=ci` succeeds
- `make image-scan IMAGE_TAG=ci` scans the built image for unfixed high/critical findings

CI does not claim to prove the full local startup/bootstrap flow. `make ready-for-adopters` is the final local release gate for this scaffold: it runs lint, Go vulnerability scanning, bootstrap smoke, Docker build, and Docker image scanning. It does not replace the [adoption checklist](docs/adoption-checklist.md) or startup-specific production hardening. See [docs/startup-readiness.md](docs/startup-readiness.md) for the exact standard.

## Auth Endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/v1/auth/register` | — | Register a new user |
| POST | `/api/v1/auth/login` | — | Login, receive tokens |
| POST | `/api/v1/auth/refresh` | — | Rotate refresh token |
| POST | `/api/v1/auth/logout` | — | Revoke refresh token |
| POST | `/api/v1/auth/forgot-password` | — | Request password reset |
| POST | `/api/v1/auth/reset-password` | — | Apply password reset |
| GET | `/api/v1/auth/me` | Bearer | Current user info |
| POST | `/api/v1/auth/change-password` | Bearer | Change password |

Health: `GET /healthz` · `GET /readyz` · `GET /metrics`

## Adding a New Domain

Use one package per product domain. Keep HTTP parsing in handlers, business rules in services, SQL in `db/queries`, and schema changes in migrations.

`internal/exampledomain` is a copyable reference package for the first domain you add. It is intentionally not mounted in the runtime router and does not add database tables. Use it to copy the handler/service/store/routes shape, then rename the package to your real domain and replace the store interface with SQLC-backed methods.

1. Create `internal/your-domain/` with `handler.go`, `service.go`, and `routes.go`.
2. Add SQL queries to `db/queries/your-domain.sql`.
3. Add schema changes with `make migrate-create name=your_domain`.
4. Run `make generate` after query changes so `db/gen` stays current.
5. Register the domain handler in `internal/server/router.go` by adding it to `Handlers` and mounting its routes.
6. Construct the service/handler in `cmd/server/main.go` next to the existing auth wiring.
7. Add focused tests beside the package first, then add integration coverage when the domain crosses auth, database, or routing boundaries.

For org-scoped routes, follow the example package's route pattern:

```go
r.Use(middleware.RequireOrgAccess(func(r *http.Request) (string, bool) {
	orgID := chi.URLParam(r, "orgID")
	return orgID, orgID != ""
}))
```

For owner/admin-only routes, add `middleware.RequireRole(auth.RoleOwner, auth.RoleAdmin)` after JWT auth.

The intended dependency direction is:

```text
cmd/server -> internal/server -> internal/<domain> -> db/gen
```

Avoid importing one product domain directly from another until there is a real shared concept. Put shared infrastructure under `internal/platform`.

## Make Targets

| Target | Description |
|--------|-------------|
| `make run` | Start server |
| `make build` | Compile to `bin/server` |
| `make test` | Unit tests |
| `make test-integration` | Integration tests (requires migrated local DB + Redis) |
| `make test-ci` | CI test gate: full package sweep plus integration tests, both with `-race` |
| `make smoke` | Focused server/auth package check |
| `make bootstrap-smoke` | Verified clean-path bootstrap check |
| `make check-adoption` | Check an initialized project for leftover scaffold ownership markers |
| `make init-template-smoke` | Verify initialization in a temporary clean copy |
| `make ready-for-adopters` | Final local release gate: lint, vuln scan, bootstrap smoke, Docker build, and image scan |
| `make vuln` | Run `govulncheck ./...` |
| `make docker-build` | Build Docker image as `$(IMAGE_NAME):$(IMAGE_TAG)` |
| `make image-scan` | Scan built Docker image with Trivy |
| `make test-all` | Both |
| `make lint` | golangci-lint |
| `make fmt` | gofmt + goimports |
| `make generate` | sqlc generate |
| `make migrate-up` | Apply migrations |
| `make migrate-down` | Roll back last migration |
| `make migrate-create name=foo` | Create new migration |
| `make docker-up` | Start postgres + redis |
| `make install-tools` | Install sqlc, goose, golangci-lint, goimports |

## Environment

Copy `.env.example` to `.env` and update values as needed. `JWT_SECRET` must not remain the example placeholder.

| Variable | Purpose |
|----------|---------|
| `PORT` | HTTP port for the API server |
| `DATABASE_URL` | Postgres connection string |
| `REDIS_URL` | Redis connection string |
| `JWT_SECRET` | HMAC signing key for access tokens |
| `JWT_EXPIRY` | Access-token lifetime, parsed by Go `time.ParseDuration` |
| `REFRESH_EXPIRY` | Refresh-token lifetime |
| `APP_BASE_URL` | Base URL used in password-reset links |
| `ALLOWED_ORIGINS` | Comma-separated browser origins allowed by CORS |
| `TRUST_PROXY_HEADERS` | Set to `true` only when the app sits behind a trusted reverse proxy or ingress |
| `TRUSTED_PROXY_CIDRS` | Comma-separated CIDR list for the proxy networks allowed to supply forwarded client IP headers |

## Release

Tag a commit to trigger your repository's release workflow after you update ownership-specific settings such as the GitHub org/user, container registry path, and image names:

```bash
git tag v1.0.0
git push origin v1.0.0
```

The scaffold can publish a GitHub Release and container image, but adopters must point that flow at their own repository and registry before using it.

Docker image targets default to `IMAGE_NAME=go-backend-scaffold`, `IMAGE_TAG=local`, and `DOCKER_BUILD_FLAGS=--pull`. Override them when checking adopter-specific builds:

```bash
make docker-build IMAGE_NAME=my-api IMAGE_TAG=dev
make image-scan IMAGE_NAME=my-api IMAGE_TAG=dev
```
