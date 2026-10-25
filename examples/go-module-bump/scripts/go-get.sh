#!/usr/bin/env bash
# For each go.mod that requires $MODULE below $VERSION, run
# `go get $MODULE@$VERSION` and `go mod tidy` in that module.
set -euo pipefail

: "${MODULE:?set MODULE}" "${VERSION:?set VERSION, e.g. v0.33.0}"

# older A B succeeds when version A sorts before B (v-prefixed semver).
older() {
  [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "${1#v}" "${2#v}" | sort -V | head -1)" = "${1#v}" ]
}

find . -name go.mod -not -path './vendor/*' -not -path '*/testdata/*' | while IFS= read -r mod; do
  current=$(awk -v m="$MODULE" '$1 == m && $2 ~ /^v/ { print $2; exit } $1 == "require" && $2 == m { print $3; exit }' "$mod")
  [ -z "$current" ] && continue
  older "$current" "$VERSION" || continue
  (
    cd "$(dirname "$mod")"
    go get "$MODULE@$VERSION"
    go mod tidy
  )
done
