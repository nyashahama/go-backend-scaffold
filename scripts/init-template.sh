#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT_DIR"

OLD_MODULE="github.com/nyashahama/go-backend-scaffold"

usage() {
  echo "usage: bash scripts/init-template.sh <new-module-path>" >&2
}

is_valid_module_path() {
  [[ "$1" =~ ^[[:alnum:]._-]+(/[[:alnum:]._-]+)+$ ]]
}

if [[ $# -ne 1 ]]; then
  usage
  exit 1
fi

NEW_MODULE="$1"

if ! is_valid_module_path "$NEW_MODULE"; then
  echo "error: module path must look like github.com/yourorg/yourapp" >&2
  exit 1
fi

if [[ ! -f go.mod ]]; then
  echo "error: go.mod not found; run this script from the scaffold repository" >&2
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
  if [[ "$file" == "./.git/"* || "$file" == "./scripts/init-template.sh" ]]; then
    continue
  fi

  if ! grep -Iq . "$file"; then
    continue
  fi

  if ! grep -qF "$OLD_MODULE" "$file"; then
    continue
  fi

  OLD_MODULE="$OLD_MODULE" NEW_MODULE="$NEW_MODULE" \
    perl -0pi -e 's/\Q$ENV{OLD_MODULE}\E/$ENV{NEW_MODULE}/g' "$file"
  FILES_UPDATED=$((FILES_UPDATED + 1))
done < <(find . -path './.git' -prune -o -type f -print0)

if [[ "$FILES_UPDATED" -eq 0 ]]; then
  echo "error: no scaffold module references were found to rewrite" >&2
  exit 1
fi

echo "updated $FILES_UPDATED files from $OLD_MODULE to $NEW_MODULE"
