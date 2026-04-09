#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

cp .env.example .env.bootstrap
export DATABASE_URL="postgres://user:change-me-local-dev@localhost:5432/scaffold?sslmode=disable"
export REDIS_URL="redis://localhost:6379"
export JWT_SECRET="bootstrap-smoke-secret-that-is-at-least-32-chars"

docker compose up -d postgres redis
make migrate-up
go test ./...
go test ./tests/integration/... -tags=integration
docker compose down -v
