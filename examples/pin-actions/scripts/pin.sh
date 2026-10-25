#!/usr/bin/env bash
# Pin every `uses: owner/repo[/path]@ref` in .github/workflows to the commit
# SHA the ref points at, keeping the ref as a comment. Local actions
# (./path) and docker:// references are left alone, as are refs that are
# already full SHAs. Needs gh on PATH.
set -euo pipefail

shopt -s nullglob
files=(.github/workflows/*.yml .github/workflows/*.yaml)
[ ${#files[@]} -eq 0 ] && exit 0

# Resolved refs are cached in a file ("repo@ref sha" per line); macOS
# ships bash 3.2, which has no associative arrays.
cache=$(mktemp)
trap 'rm -f "$cache"' EXIT
resolve() {
  local key="$1@$2" sha
  sha=$(awk -v k="$key" '$1 == k { print $2; exit }' "$cache")
  if [ -z "$sha" ]; then
    sha=$(gh api "repos/$1/commits/$2" --jq .sha)
    printf '%s %s\n' "$key" "$sha" >>"$cache"
  fi
  printf '%s' "$sha"
}

re='^([[:space:]]*-?[[:space:]]*uses:[[:space:]]*)([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)(/[^@[:space:]]*)?@([^[:space:]#]+)(.*)$'
for f in "${files[@]}"; do
  out=$(mktemp)
  while IFS= read -r line || [ -n "$line" ]; do
    if [[ $line =~ $re ]]; then
      prefix=${BASH_REMATCH[1]} repo=${BASH_REMATCH[2]} sub=${BASH_REMATCH[3]} ref=${BASH_REMATCH[4]}
      if [[ ! $ref =~ ^[0-9a-f]{40}$ ]]; then
        sha=$(resolve "$repo" "$ref")
        line="${prefix}${repo}${sub}@${sha} # ${ref}"
      fi
    fi
    printf '%s\n' "$line" >>"$out"
  done <"$f"
  mv "$out" "$f"
done
