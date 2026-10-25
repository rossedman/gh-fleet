#!/usr/bin/env bash
# Replace every occurrence of $FROM with $TO in tracked text files. The
# match is literal, so dots and slashes need no escaping. Binary files and
# anything git ignores are left alone.
set -euo pipefail

: "${FROM:?set FROM}"
: "${TO?set TO}"

# git grep exits 1 when nothing matches; that is a no-op, not a failure.
{ git grep -lIzF -e "$FROM" -- . || true; } | while IFS= read -r -d '' f; do
  perl -pi -e 's/\Q$ENV{FROM}\E/$ENV{TO}/g' "$f"
done
