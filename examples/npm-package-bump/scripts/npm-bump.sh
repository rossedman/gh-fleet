#!/usr/bin/env bash
# For each package.json listing $PACKAGE in dependencies or
# devDependencies, set it to $RANGE and refresh the lockfile. Needs jq and
# npm. Install scripts never run.
set -euo pipefail

: "${PACKAGE:?set PACKAGE}" "${RANGE:?set RANGE, e.g. ^1.7.9}"

find . -name package.json -not -path '*/node_modules/*' -not -path './.git/*' | while IFS= read -r pkg; do
  section=$(jq -r --arg p "$PACKAGE" '
    if .dependencies[$p] then "dependencies"
    elif .devDependencies[$p] then "devDependencies"
    else empty end' "$pkg")
  [ -z "$section" ] && continue
  [ "$(jq -r --arg p "$PACKAGE" --arg s "$section" '.[$s][$p]' "$pkg")" = "$RANGE" ] && continue
  (
    cd "$(dirname "$pkg")"
    flag=--save-prod
    [ "$section" = devDependencies ] && flag=--save-dev
    npm install "$PACKAGE@$RANGE" "$flag" --package-lock-only --ignore-scripts --no-audit --no-fund
  )
done
