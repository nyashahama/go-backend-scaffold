#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
TMP_DIR="$(mktemp -d "${TMPDIR:-/tmp}/init-template-smoke.XXXXXX")"

cleanup() {
  rm -rf "$TMP_DIR"
}
trap cleanup EXIT

cd "$ROOT_DIR"
git ls-files -z | tar --null -T - -cf - | tar -x -C "$TMP_DIR"

cd "$TMP_DIR"
git init -q
git config user.email "init-smoke@example.com"
git config user.name "Init Smoke"
git add .
git commit -qm "initial scaffold"

bash scripts/init-template.sh github.com/example/acme-api
make check-adoption
go test ./...
