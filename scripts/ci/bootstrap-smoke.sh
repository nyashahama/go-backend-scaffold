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
