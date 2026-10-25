#!/usr/bin/env bash
# Write .github/CODEOWNERS with `* @<owner>/$FLEET_VAR_TEAM` unless a
# CODEOWNERS file already exists.
set -euo pipefail

: "${FLEET_VAR_TEAM:?set vars.default.team in the campaign}"
owner=${FLEET_REPO%%/*}

for f in .github/CODEOWNERS CODEOWNERS docs/CODEOWNERS; do
  [ -e "$f" ] && exit 0
done
mkdir -p .github
printf '# Owners are requested for review on every pull request.\n* @%s/%s\n' "$owner" "$FLEET_VAR_TEAM" >.github/CODEOWNERS
