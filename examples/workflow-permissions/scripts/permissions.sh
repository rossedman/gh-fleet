#!/usr/bin/env bash
# Insert `permissions: contents: read` above the top-level `jobs:` key of
# every workflow without a top-level `permissions:` key. Plain awk, so
# comments and formatting are kept exactly.
set -euo pipefail

shopt -s nullglob
for f in .github/workflows/*.yml .github/workflows/*.yaml; do
  grep -q '^permissions:' "$f" && continue
  grep -q '^jobs:' "$f" || continue
  awk '/^jobs:/ && !done { print "permissions:\n  contents: read\n"; done = 1 } { print }' "$f" >"$f.tmp"
  mv "$f.tmp" "$f"
done
