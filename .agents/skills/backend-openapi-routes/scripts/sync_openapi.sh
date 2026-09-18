#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/../../../.." && pwd)"
BACKEND_DIR="$REPO_ROOT/backend"

if [[ ! -f "$BACKEND_DIR/Makefile" ]]; then
  printf 'Cannot find backend Makefile under %s\n' "$BACKEND_DIR" >&2
  exit 1
fi

printf 'Generating Swaggo OpenAPI artifacts, then ReDoc...\n'
(cd "$BACKEND_DIR" && make openapi)

printf 'Validating generated artifacts...\n'
jq empty "$BACKEND_DIR/docs/swagger.json"
test -s "$BACKEND_DIR/docs/redoc.html"

if ! git -C "$REPO_ROOT" diff --check -- . ':(exclude)backend/docs/redoc.html'; then
  printf 'Source or OpenAPI changes contain whitespace errors.\n' >&2
  exit 1
fi

printf 'OpenAPI and ReDoc artifacts are synchronized.\n'
