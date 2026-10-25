#!/usr/bin/env bash
# Write .github/CODEOWNERS with `* $OWNER` unless a CODEOWNERS file exists.
set -euo pipefail

: "${OWNER:?set OWNER, e.g. @acme/platform}"

for f in .github/CODEOWNERS CODEOWNERS docs/CODEOWNERS; do
  [ -e "$f" ] && exit 0
done
mkdir -p .github
printf '# Owners are requested for review on every pull request.\n* %s\n' "$OWNER" >.github/CODEOWNERS
