#!/usr/bin/env bash

set -euo pipefail

project_root="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
env_file="${project_root}/.env"

if [[ ! -f "${env_file}" ]]; then
  echo "Missing ${env_file}. Copy .env.example to .env and fill in JWT_SECRET."
  exit 1
fi

set -a
# shellcheck disable=SC1090
source "${env_file}"
set +a

: "${DATABASE_URL:?DATABASE_URL must be set in .env}"
: "${JWT_SECRET:?JWT_SECRET must be set in .env}"
export NEXT_PUBLIC_API_URL="${NEXT_PUBLIC_API_URL:-http://localhost:8080}"

cleanup() {
  echo "Stopping frontend and backend..."
  kill "${backend_pid}" "${frontend_pid}" 2>/dev/null || true
  wait "${backend_pid}" "${frontend_pid}" 2>/dev/null || true
}

trap cleanup EXIT INT TERM

echo "Starting backend on ${HTTP_ADDR:-:8080}..."
(
  cd "${project_root}/backend"
  go run ./cmd/api
) &
backend_pid=$!

echo "Starting frontend on http://localhost:3000..."
(
  cd "${project_root}/frontend"
  npm run dev
) &
frontend_pid=$!

while kill -0 "${backend_pid}" 2>/dev/null && kill -0 "${frontend_pid}" 2>/dev/null; do
  sleep 1
done

exit_code=0
wait "${backend_pid}" || exit_code=$?
wait "${frontend_pid}" || exit_code=$?
exit "${exit_code}"
