#!/usr/bin/env bash
# Seed script — add your dev seed data here.
# Example: register a test user via the API.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"
payload='{"email":"admin@example.com","password":"DevPassword123!","full_name":"Dev Admin"}'

echo "Seeding development data..."

response="$(curl -sS -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "$payload" \
  -w '\n%{http_code}')"

status="${response##*$'\n'}"
body="${response%$'\n'*}"

case "$status" in
  201)
    if command -v jq >/dev/null 2>&1; then
      printf '%s\n' "$body" | jq .
    else
      printf '%s\n' "$body"
    fi
    ;;
  409)
    echo "Seed user already exists."
    ;;
  *)
    echo "Seed failed with HTTP $status" >&2
    printf '%s\n' "$body" >&2
    exit 1
    ;;
esac

echo "Done."
