#!/usr/bin/env bash
# Delete each path in $REMOVE that exists. Missing paths are skipped, so a
# repo without any of them is a no-op.
set -euo pipefail

: "${REMOVE:?set REMOVE to a space-separated list of paths}"

for p in $REMOVE; do
  case "$p" in
  /* | *..*)
    echo "refusing to remove $p: use a path inside the repo" >&2
    exit 1
    ;;
  esac
  rm -rf -- "./$p"
done
