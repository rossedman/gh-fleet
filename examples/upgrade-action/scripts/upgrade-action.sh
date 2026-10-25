#!/usr/bin/env bash
# Replace `uses: $ACTION@$FROM` with `uses: $ACTION@$TO` in workflow and
# action files. Matches the whole ref, so v3 does not match v3.1 or v30.
set -euo pipefail

: "${ACTION:?set ACTION}" "${FROM:?set FROM}" "${TO:?set TO}"

[ -d .github ] || exit 0
find .github -type f \( -name '*.yml' -o -name '*.yaml' \) |
  while IFS= read -r f; do
    perl -pi -e 's/(uses:\s*["\x27]?\Q$ENV{ACTION}\E(?:\/[^@\s]*)?)\@\Q$ENV{FROM}\E(?=["\x27\s]|$)/$1\@$ENV{TO}/g' "$f"
  done
