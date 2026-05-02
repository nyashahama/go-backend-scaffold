# Phase 1 Readiness Gate Design

## Goal

Make the scaffold truthfully bootable and verifiable from a normal checkout. After this phase, a maintainer or adopter should be able to run the documented bootstrap and readiness commands without relying on hidden local database state, exported shell variables, or misleading documentation.

## Scope

This phase covers the first reliability layer of the scaffold:

1. The final readiness gate.
2. Local `.env` and Makefile behavior.
3. Integration test environment defaults.
4. Docker build context hygiene.
5. Seed script correctness.
6. README and readiness documentation claims.

This phase does not add new product features, new auth behavior, CI security scanning, OpenAPI docs, example domains, trusted proxy configuration, or broader runtime hardening. Those belong in later phases.

## Current Findings

The scaffold already has strong fundamentals: unit tests pass with race detection, the isolated bootstrap smoke target passes, and the Docker image builds. The problem is that the documented final release gate does not compose those checks correctly.

`make ready-for-adopters` currently calls `make test-ci` before `make bootstrap-smoke`. `test-ci` runs integration tests directly against local `localhost:5432` and `localhost:6379`, so the final gate fails unless the developer already has a migrated database with matching credentials. That contradicts the readiness docs, which say the gate proves a clean-path bootstrap.

The local quickstart has a similar issue. It tells adopters to copy `.env.example` and then run `make migrate-up`, but the Makefile does not load `.env`, so those values are not available to `goose` unless the user manually exports them.

There are also smaller trust issues:

- The integration test fallback Postgres password does not match `.env.example` or `docker-compose.yml`.
- The README says "production-ready", while the adoption checklist correctly says this is a starting point.
- There is no tracked `.dockerignore`, so unnecessary and sensitive local files can enter the Docker build context.
- `scripts/seed.sh` uses a password that fails the scaffold's current password rules and masks real API failures as missing tooling.

## Design

### 1. Readiness Gate

`make ready-for-adopters` will become self-contained. It will run:

1. `make lint`
2. `make bootstrap-smoke`
3. `docker build -t go-backend-scaffold:ready .`

`bootstrap-smoke` will run the unit and integration tests with race detection after creating isolated Docker Compose services and applying migrations. That means the final gate still proves tests, integration behavior, clean bootstrap, and Docker build, but it no longer depends on whichever database happens to be listening on local port 5432.

`make test-ci` can remain the direct CI target for environments that have already provisioned Postgres and Redis, such as GitHub Actions services. The docs must state that distinction clearly.

### 2. Makefile Environment Loading

The Makefile will load `.env` when present and export those values to child commands. This makes the documented local bootstrap work after:

```bash
cp .env.example .env
```

The Makefile should not require `.env` for every target. Targets such as `test`, `build`, and `bootstrap-smoke` must continue to work without it. `bootstrap-smoke` already creates and exports its own `.env.bootstrap` values.

### 3. Integration Test Defaults

The fallback database URL in `tests/integration/testhelpers_test.go` will be aligned with the Compose and `.env.example` credentials:

```text
postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable
```

This does not replace explicit environment variables. It only makes the no-env fallback match the repo's own local infrastructure.

### 4. Docker Build Context Hygiene

Add a tracked `.dockerignore` that excludes local secrets, Git metadata, worktrees, build output, test artifacts, editor files, and temporary env files while preserving `.env.example`.

This keeps Docker builds faster and prevents accidental inclusion of local secrets or unrelated worktree contents in image build context.

### 5. Seed Script

Update `scripts/seed.sh` so the example password satisfies the current auth password policy. The script should also distinguish between:

- successful user creation
- "already exists" conflicts
- real API failures
- optional `jq` formatting

It should not hide a failed registration behind a generic "curl or jq not available" message.

### 6. Documentation Claims

README and readiness docs will use bounded language:

- "production-oriented starting point" or "startup backend scaffold"
- not "production-ready drop-in"

The README quickstart should describe the actual local path:

1. Copy `.env.example` to `.env`.
2. Replace `JWT_SECRET`.
3. Start Docker services.
4. Run migrations.
5. Run tests.
6. Start the server.

The readiness docs should define exactly what `make ready-for-adopters` proves and what it still does not prove.

## Error Handling

- If `.env` is absent, non-runtime Make targets should continue normally.
- If `DATABASE_URL` is missing for migration targets, `goose` may still fail, but the README path should avoid that for normal adopters.
- If `bootstrap-smoke` fails, cleanup must still remove temporary Compose resources and `.env.bootstrap`.
- If the seed API call fails for an unexpected status, the script should print the response body and exit non-zero.

## Testing Strategy

Use existing commands as the verification surface:

- `go test ./... -race`
- `make bootstrap-smoke`
- `docker build -t go-backend-scaffold:assessment .`
- `make ready-for-adopters`

For local quickstart validation:

- `cp .env.example .env`
- edit `JWT_SECRET`
- `make docker-up`
- `make migrate-up`
- `go test ./tests/integration/... -v -race -tags=integration`
- `make docker-down`

The final success condition is that `make ready-for-adopters` passes without relying on a pre-existing local database or Redis instance.

## Success Criteria

Phase 1 is complete when:

- `make ready-for-adopters` passes from a normal checkout with Docker available.
- The gate does not depend on local port 5432 or 6379 containing the right pre-migrated services.
- `.env` copied from `.env.example` is enough for Makefile migration and run targets after the user replaces `JWT_SECRET`.
- Integration fallback credentials match Compose defaults.
- Docker build context excludes local secrets and repository metadata.
- `scripts/seed.sh` uses a valid password and reports real API failures accurately.
- README and readiness docs no longer overclaim production readiness.
