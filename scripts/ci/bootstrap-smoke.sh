#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

find_free_port() {
  local port
  for port in "$@"; do
    if ! ss -ltn "( sport = :$port )" | grep -q ":$port"; then
      printf '%s\n' "$port"
      return 0
    fi
  done

  echo "failed to find a free port" >&2
  return 1
}

cleanup() {
  docker compose down -v
  rm -f .env.bootstrap
}

trap cleanup EXIT

cp .env.example .env.bootstrap
export COMPOSE_PROJECT_NAME="bootstrap-smoke"
export POSTGRES_PORT="$(find_free_port 15432 25432 35432 45432)"
export REDIS_PORT="$(find_free_port 16379 26379 36379 46379)"
export DATABASE_URL="postgres://user:change-me-local-dev@localhost:${POSTGRES_PORT}/scaffold?sslmode=disable"
export REDIS_URL="redis://localhost:${REDIS_PORT}"
export JWT_SECRET="bootstrap-smoke-secret-that-is-at-least-32-chars"

docker compose up -d --wait postgres redis
make migrate-up
go test ./...
go test ./tests/integration/... -tags=integration
