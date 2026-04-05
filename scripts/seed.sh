#!/usr/bin/env bash
# Seed script — add your dev seed data here.
# Example: register a test user via the API.
set -euo pipefail

BASE_URL="${BASE_URL:-http://localhost:8080}"

echo "Seeding development data..."

curl -sf -X POST "$BASE_URL/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"dev-password-123","full_name":"Dev Admin"}' \
  | jq . || echo "(curl or jq not available — skipping)"

echo "Done."
