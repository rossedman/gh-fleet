#!/usr/bin/env bash
# Rewrite `FROM $IMAGE:$FROM_TAG[-variant]` to `$TO_TAG` in every
# Dockerfile (Dockerfile, *.Dockerfile, Dockerfile.*).
set -euo pipefail

: "${IMAGE:?set IMAGE}" "${FROM_TAG:?set FROM_TAG}" "${TO_TAG:?set TO_TAG}"

find . -path ./.git -prune -o -type f \( -name Dockerfile -o -name '*.Dockerfile' -o -name 'Dockerfile.*' \) -print |
  while IFS= read -r f; do
    perl -pi -e 's/^(\s*FROM\s+(?:--platform=\S+\s+)?)\Q$ENV{IMAGE}\E:\Q$ENV{FROM_TAG}\E(?=[-\s@]|$)/$1$ENV{IMAGE}:$ENV{TO_TAG}/i' "$f"
  done
