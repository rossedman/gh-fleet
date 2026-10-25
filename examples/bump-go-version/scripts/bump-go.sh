#!/usr/bin/env bash
# Raise the go directive in every go.mod to $GO_VERSION (never lowering
# it) and update go-version pins in workflows.
set -euo pipefail

: "${GO_VERSION:?set GO_VERSION, e.g. 1.27.3}"

# newer A B succeeds when version A is newer than B.
newer() {
  [ "$1" != "$2" ] && [ "$(printf '%s\n%s\n' "$1" "$2" | sort -V | tail -1)" = "$1" ]
}

while IFS= read -r mod; do
  current=$(sed -n 's/^go \([0-9.]*\).*/\1/p' "$mod")
  if [ -z "$current" ] || newer "$GO_VERSION" "$current"; then
    sed -i.bak -E "s/^go [0-9.]+.*/go $GO_VERSION/" "$mod"
    sed -i.bak -E "/^toolchain /d" "$mod"
    rm -f "$mod.bak"
  fi
done < <(find . -name go.mod -not -path './vendor/*' -not -path '*/testdata/*')

shopt -s nullglob
for f in .github/workflows/*.yml .github/workflows/*.yaml; do
  sed -i.bak -E "s/(go-version:[[:space:]]*['\"]?)[0-9]+\.[0-9]+(\.[0-9]+)?(['\"]?)/\1${GO_VERSION}\3/" "$f"
  rm -f "$f.bak"
done
