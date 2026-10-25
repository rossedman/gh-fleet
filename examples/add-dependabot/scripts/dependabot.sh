#!/usr/bin/env bash
# Write .github/dependabot.yml for the ecosystems present in the repo,
# unless one already exists.
set -euo pipefail

cfg=.github/dependabot.yml
[ -e "$cfg" ] || [ -e .github/dependabot.yaml ] && exit 0

ecosystems=()
[ -f go.mod ] && ecosystems+=("gomod:build(deps)")
[ -f package.json ] && ecosystems+=("npm:build(deps)")
{ [ -f pyproject.toml ] || [ -f requirements.txt ]; } && ecosystems+=("pip:build(deps)")
[ -f Dockerfile ] && ecosystems+=("docker:build(deps)")
[ -d .github/workflows ] && ecosystems+=("github-actions:ci(deps)")
[ ${#ecosystems[@]} -eq 0 ] && exit 0

mkdir -p .github
{
  echo "version: 2"
  echo "updates:"
  for e in "${ecosystems[@]}"; do
    name=${e%%:*} prefix=${e#*:}
    cat <<YAML
  - package-ecosystem: $name
    directory: /
    schedule:
      interval: weekly
    groups:
      minor-and-patch:
        update-types: [minor, patch]
    commit-message:
      prefix: "$prefix"
YAML
  done
} >"$cfg"
