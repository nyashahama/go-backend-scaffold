#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

SMOKE_ENV_FILE=".env.bootstrap"
SMOKE_COMPOSE_PROJECT="go-backend-scaffold-smoke-$(date +%s)-$$"

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

POSTGRES_PORT="${POSTGRES_PORT:-$(find_free_port 15432 25432 35432 45432)}"
REDIS_PORT="${REDIS_PORT:-$(find_free_port 16379 26379 36379 46379)}"
DATABASE_URL="postgres://user:change-me-local-dev@localhost:${POSTGRES_PORT}/scaffold?sslmode=disable"
REDIS_URL="redis://localhost:${REDIS_PORT}"
JWT_SECRET="bootstrap-smoke-secret-that-is-at-least-32-chars"

export POSTGRES_PORT REDIS_PORT DATABASE_URL REDIS_URL JWT_SECRET
cp .env.example "$SMOKE_ENV_FILE"

cat >>"$SMOKE_ENV_FILE" <<EOF
COMPOSE_PROJECT_NAME=${SMOKE_COMPOSE_PROJECT}
POSTGRES_PORT=${POSTGRES_PORT}
REDIS_PORT=${REDIS_PORT}
DATABASE_URL=${DATABASE_URL}
REDIS_URL=${REDIS_URL}
JWT_SECRET=${JWT_SECRET}
EOF

cleanup() {
  docker compose --env-file "$SMOKE_ENV_FILE" down -v
  rm -f "$SMOKE_ENV_FILE"
}
trap cleanup EXIT

docker compose --env-file "$SMOKE_ENV_FILE" up -d --wait postgres redis
make migrate-up
go test ./...
go test ./tests/integration/... -tags=integration
