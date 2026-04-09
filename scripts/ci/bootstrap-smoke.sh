#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
cd "$ROOT_DIR"

SMOKE_ENV_FILE=".env.bootstrap"
SMOKE_COMPOSE_PROJECT="go-backend-scaffold-smoke-$(date +%s)-$$"

find_free_port() {
  local helper
  local status
  helper="$(mktemp "${TMPDIR:-/tmp}/bootstrap-port-check.XXXXXX.go")"
  cat >"$helper" <<'EOF'
package main

import (
	"fmt"
	"net"
	"os"
)

func main() {
	for _, port := range os.Args[1:] {
		ln, err := net.Listen("tcp", "127.0.0.1:"+port)
		if err != nil {
			continue
		}
		_ = ln.Close()
		fmt.Println(port)
		return
	}

	fmt.Fprintln(os.Stderr, "failed to find a free port")
	os.Exit(1)
}
EOF
  go run "$helper" "$@" || status=$?
  rm -f "$helper"
  return "${status:-0}"
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
