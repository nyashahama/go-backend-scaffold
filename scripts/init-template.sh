#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

OLD_MODULE="github.com/nyashahama/go-backend-scaffold"

usage() {
  echo "usage: bash scripts/init-template.sh <new-module-path>" >&2
  echo "run this once on a fresh, unmodified clone before making project-specific edits" >&2
}

is_valid_module_path() {
  [[ "$1" =~ ^[[:alnum:]._-]+(/[[:alnum:]._-]+)+$ ]]
}

replace_in_file() {
  local file="$1"
  local oldValue="${2:-$OLD_MODULE}"
  local newValue="${3:-$NEW_MODULE}"
  local helper

  helper="$(mktemp "${TMPDIR:-/tmp}/init-template.XXXXXX.go")"
  trap 'rm -f "$helper"' RETURN
  cat >"$helper" <<'EOF'
package main

import (
	"fmt"
	"os"
	"strings"
)

func main() {
	path := os.Getenv("TARGET_FILE")
	oldValue := os.Getenv("OLD_VALUE")
	newValue := os.Getenv("NEW_VALUE")

	if path == "" || oldValue == "" || newValue == "" {
		fmt.Fprintln(os.Stderr, "missing replacement context")
		os.Exit(1)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read %s: %v\n", path, err)
		os.Exit(1)
	}

	updated := strings.ReplaceAll(string(data), oldValue, newValue)
	if updated == string(data) {
		fmt.Fprintln(os.Stderr, "replacement target not found")
		os.Exit(1)
	}

	if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", path, err)
		os.Exit(1)
	}
}
EOF
  OLD_VALUE="$oldValue" NEW_VALUE="$newValue" TARGET_FILE="$file" go run "$helper"
}

replace_if_present() {
  local file="$1"
  local oldValue="$2"
  local newValue="$3"

  if [[ -f "$file" ]] && grep -qF "$oldValue" "$file"; then
    replace_in_file "$file" "$oldValue" "$newValue"
  fi
}

if ! command -v git >/dev/null 2>&1; then
  echo "error: git is required to initialize this scaffold" >&2
  exit 1
fi

if ! command -v go >/dev/null 2>&1; then
  echo "error: go is required to rewrite module references in this scaffold" >&2
  exit 1
fi

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

NEW_MODULE="$1"
APP_NAME="${NEW_MODULE##*/}"
TOKEN_ISSUER="${APP_NAME}"
TOKEN_AUDIENCE="${APP_NAME}-api"

if ! is_valid_module_path "$NEW_MODULE"; then
  echo "error: module path must look like github.com/yourorg/yourapp" >&2
  exit 1
fi

if [[ ! -f go.mod ]]; then
  echo "error: go.mod not found; run this script from the scaffold repository" >&2
  exit 1
fi

if ! git rev-parse --show-toplevel >/dev/null 2>&1; then
  echo "error: this script must run inside a git clone of the scaffold" >&2
  exit 1
fi

if [[ -n "$(git status --porcelain --untracked-files=all)" ]]; then
  echo "error: initialization requires a clean worktree" >&2
  echo "error: run this once on a fresh clone before editing files" >&2
  exit 1
fi

CURRENT_MODULE="$(awk '/^module / { print $2; exit }' go.mod)"
if [[ -z "$CURRENT_MODULE" ]]; then
  echo "error: could not determine the current module path from go.mod" >&2
  exit 1
fi

if [[ "$CURRENT_MODULE" != "$OLD_MODULE" ]]; then
  echo "error: expected go.mod module path $OLD_MODULE, found $CURRENT_MODULE" >&2
  echo "error: this script is only for the unmodified scaffold clone" >&2
  exit 1
fi

FILES_UPDATED=0

while IFS= read -r -d '' file; do
  if [[ "$file" == "scripts/init-template.sh" ]]; then
    continue
  fi

  replace_in_file "$file"
  FILES_UPDATED=$((FILES_UPDATED + 1))
done < <(git grep -lzF "$OLD_MODULE" -- . ':(exclude)scripts/init-template.sh')

replace_if_present "$ROOT_DIR/.env.example" "__scaffold_issuer__" "$TOKEN_ISSUER"
replace_if_present "$ROOT_DIR/.env.example" "__scaffold_audience__" "$TOKEN_AUDIENCE"
replace_if_present "$ROOT_DIR/.env.example" "go-backend-scaffold-api" "$TOKEN_AUDIENCE"
replace_if_present "$ROOT_DIR/.env.example" "go-backend-scaffold" "$APP_NAME"
replace_if_present "$ROOT_DIR/Makefile" "go-backend-scaffold" "$APP_NAME"
replace_if_present "$ROOT_DIR/.github/workflows/ci.yml" "go-backend-scaffold" "$APP_NAME"
replace_if_present "$ROOT_DIR/.github/workflows/release.yml" "go-backend-scaffold" "$APP_NAME"
replace_if_present "$ROOT_DIR/scripts/ci/bootstrap-smoke.sh" "go-backend-scaffold-smoke" "${APP_NAME}-smoke"

if [[ "$FILES_UPDATED" -eq 0 ]]; then
  echo "error: no scaffold module references were found to rewrite" >&2
  exit 1
fi

echo "updated $FILES_UPDATED files from $OLD_MODULE to $NEW_MODULE"
