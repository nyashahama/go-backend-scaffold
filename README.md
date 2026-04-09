# go-backend-scaffold

A production-ready Go backend scaffold. Clone it, rename the module, and build your next API.

**Stack:** chi · pgx/v5 · sqlc · goose · JWT · Redis · Prometheus · slog · Docker · GitHub Actions

## Quickstart

### 1. Clone and rename

```bash
git clone https://github.com/nyashahama/go-backend-scaffold.git my-api
cd my-api
find . -type f -name "*.go" -exec sed -i 's|github.com/nyashahama/go-backend-scaffold|github.com/yourname/my-api|g' {} +
sed -i 's|github.com/nyashahama/go-backend-scaffold|github.com/yourname/my-api|g' go.mod
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
```

The server starts on `http://localhost:8080`.

For a full containerized stack, including the backend container, run:

```bash
docker compose --profile full up --build
```

## Quality Gates

GitHub Actions verifies the startup path promised in this README:

- database migrations apply cleanly against a fresh Postgres service
- `make test-ci` passes, which runs the repository test sweep plus integration tests with race detection
- `docker build -t go-backend-scaffold:ci .` succeeds

CI does not currently run `make bootstrap-smoke`; use that locally when you want a clean-path clone-and-boot check.

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

1. Create `internal/your-domain/` with `handler.go`, `service.go`, `routes.go`
2. Add queries to `db/queries/your-domain.sql` and run `make generate`
3. Add a migration in `db/migrations/` with `make migrate-create name=your_domain`
4. Register handler in `internal/server/router.go` (add to `Handlers` struct and mount routes)
5. Wire the service in `cmd/server/main.go`

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

## Release

Tag a commit to trigger the release workflow:

```bash
git tag v1.0.0
git push origin v1.0.0
```

This builds the binary, attaches it to the GitHub Release, and pushes a Docker image to `ghcr.io/nyashahama/go-backend-scaffold`.
