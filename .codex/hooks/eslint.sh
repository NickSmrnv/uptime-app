#!/usr/bin/env bash

set -euo pipefail

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
frontend_root="$repo_root/frontend"

changed_files=()
while IFS= read -r path; do
    case "$path" in
        frontend/*)
            case "$path" in
                *.js|*.jsx|*.ts|*.tsx|*.mjs|*.cjs|*.mts|*.cts)
                    changed_files+=("${path#frontend/}")
                    ;;
            esac
            ;;
    esac
done < <(
    {
        git -C "$repo_root" diff --name-only --diff-filter=AM -- 'frontend/*'
        git -C "$repo_root" diff --cached --name-only --diff-filter=AM -- 'frontend/*'
        git -C "$repo_root" ls-files --others --exclude-standard -- 'frontend/*'
    } | sort -u
)

if [ "${#changed_files[@]}" -eq 0 ]; then
    exit 0
fi

eslint="$frontend_root/node_modules/.bin/eslint"
if [ ! -x "$eslint" ]; then
    printf '%s\n' "ESLint is unavailable: run npm install in frontend/ before editing frontend files." >&2
    exit 1
fi

cd "$frontend_root"
"$eslint" -- "${changed_files[@]}"
