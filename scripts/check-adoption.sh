#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

patterns=(
  "github.com/nyashahama/go-backend-scaffold"
  "go-backend-scaffold-api"
  "go-backend-scaffold-smoke"
  "go-backend-scaffold"
  "__scaffold_issuer__"
  "__scaffold_audience__"
)

found=0
for pattern in "${patterns[@]}"; do
  if matches="$(git grep -nF "$pattern" -- . ':(exclude)scripts/init-template.sh' ':(exclude)scripts/check-adoption.sh' ':(exclude)docs' || true)" && [[ -n "$matches" ]]; then
    echo "found stale scaffold marker: $pattern"
    echo "$matches"
    found=1
  fi
done

if [[ "$found" -ne 0 ]]; then
  echo "adoption check failed: run scripts/init-template.sh and update remaining ownership labels" >&2
  exit 1
fi

echo "adoption check passed"
