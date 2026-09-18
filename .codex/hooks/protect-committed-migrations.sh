#!/bin/sh

set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/../.." && pwd)
payload=$(cat)

if ! patch_command=$(printf '%s' "$payload" | jq -r '.tool_input.command // empty'); then
    exit 0
fi

path_list=$(mktemp "${TMPDIR:-/tmp}/protect-migrations.XXXXXX")
trap 'rm -f "$path_list"' EXIT
printf '%s\n' "$patch_command" | sed -nE \
    's/^\*\*\* (Update File|Delete File|Move to): (.*)$/\2/p' > "$path_list"

blocked_paths=""
max_version=0
for migration in "$repo_root"/backend/migrations/*.sql; do
    [ -f "$migration" ] || continue
    filename=${migration##*/}
    version=${filename%%_*}
    case "$version" in
        ''|*[!0-9]*) continue ;;
    esac
    version_number=$(printf '%s' "$version" | sed 's/^0*//')
    [ -n "$version_number" ] || version_number=0
    [ "$version_number" -gt "$max_version" ] && max_version=$version_number
done
next_version=$(printf '%06d' $((max_version + 1)))

while IFS= read -r path; do
    path=${path#\"}
    path=${path%\"}

    case "$path" in
        backend/migrations/*)
            case "$path" in
                *..*|/*)
                    :
                    ;;
                *)
                    if git -C "$repo_root" cat-file -e "HEAD:$path" 2>/dev/null; then
                        [ -n "$blocked_paths" ] && blocked_paths="$blocked_paths, "
                        blocked_paths="${blocked_paths}${path}"
                    fi
                    ;;
            esac
            ;;
    esac
done < "$path_list"

[ -n "$blocked_paths" ] || exit 0

jq -n --arg paths "$blocked_paths" --arg next_version "$next_version" '{
    hookSpecificOutput: {
        hookEventName: "PreToolUse",
        permissionDecision: "deny",
        permissionDecisionReason: ("Editing or deleting committed migrations is blocked: " + $paths + ". Add the change in a new migration, for example backend/migrations/" + $next_version + "_describe_change.sql.")
    }
}'
