#!/usr/bin/env bash
# Refresh package-lock.json in each directory the Starlark step changed
# (FLEET_VAR_DIRS, comma-separated). Install scripts never run.
set -euo pipefail

[ -n "${FLEET_VAR_DIRS:-}" ] || exit 0

IFS=, read -r -a dirs <<<"$FLEET_VAR_DIRS"
for dir in "${dirs[@]}"; do
  [ -f "$dir/package-lock.json" ] || continue
  (cd "$dir" && npm install --package-lock-only --ignore-scripts --no-audit --no-fund)
done
