#!/usr/bin/env bash
# Copy every file under ../files into the repo root, overwriting existing
# copies. Repos that already match come out as no-ops.
set -euo pipefail

src="$(cd "$(dirname "$0")/../files" && pwd)"
cp -R "$src/." .
