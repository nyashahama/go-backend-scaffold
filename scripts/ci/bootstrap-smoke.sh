#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

SMOKE_ENV_FILE=".env.bootstrap"
SMOKE_COMPOSE_PROJECT="go-backend-scaffold-smoke-$(date +%s)-$$"
cp .env.example "$SMOKE_ENV_FILE"

cat >>"$SMOKE_ENV_FILE" <<EOF
COMPOSE_PROJECT_NAME=${SMOKE_COMPOSE_PROJECT}
POSTGRES_PORT=${POSTGRES_PORT:-15432}
REDIS_PORT=${REDIS_PORT:-16379}
DATABASE_URL=postgres://user:change-me-local-dev@localhost:${POSTGRES_PORT:-15432}/scaffold?sslmode=disable
REDIS_URL=redis://localhost:${REDIS_PORT:-16379}
JWT_SECRET=bootstrap-smoke-secret-that-is-at-least-32-chars
EOF

while IFS= read -r line || [ -n "$line" ]; do
  case "$line" in
    ''|\#*)
      continue
      ;;
  esac

  export "$line"
done < "$SMOKE_ENV_FILE"

cleanup() {
  docker compose --env-file "$SMOKE_ENV_FILE" down -v
  rm -f "$SMOKE_ENV_FILE"
}
trap cleanup EXIT

docker compose --env-file "$SMOKE_ENV_FILE" up -d --wait postgres redis
make migrate-up
go test ./...
go test ./tests/integration/... -tags=integration
