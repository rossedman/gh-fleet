#!/usr/bin/env bash
# Append each line of $ENTRIES to .gitignore if it is not already present,
# under a comment header. A repo that has them all is a no-op.
set -euo pipefail

: "${ENTRIES:?set ENTRIES, one pattern per line}"

touch .gitignore
missing=()
while IFS= read -r line; do
  [ -z "$line" ] && continue
  grep -qxF -- "$line" .gitignore || missing+=("$line")
done <<<"$ENTRIES"
[ ${#missing[@]} -eq 0 ] && exit 0

# End the existing content with a newline, then leave a blank line
# before the new block.
sep=""
if [ -s .gitignore ]; then
  sep=$'\n'
  [ "$(tail -c1 .gitignore)" != "" ] && sep=$'\n\n'
fi
{
  printf '%s' "$sep"
  echo "# Local environment files"
  printf '%s\n' "${missing[@]}"
} >>.gitignore
