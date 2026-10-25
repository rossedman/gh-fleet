"""Detect package ecosystems and write a Dependabot configuration."""

# Each ecosystem and the root files that mean the repo uses it.
DETECT = [
    ("gomod", ["go.mod"]),
    ("npm", ["package.json"]),
    ("pip", ["pyproject.toml", "requirements.txt", "setup.py"]),
    ("cargo", ["Cargo.toml"]),
    ("bundler", ["Gemfile"]),
    ("composer", ["composer.json"]),
    ("maven", ["pom.xml"]),
    ("gradle", ["build.gradle", "build.gradle.kts"]),
    ("docker", ["Dockerfile"]),
    ("github-actions", [".github/workflows"]),
]

def find(repo):
    """Returns the path of the repo's Dependabot config, or None."""
    for path in [".github/dependabot.yml", ".github/dependabot.yaml"]:
        if repo.exists(path):
            return path
    return None

def ecosystems(repo):
    """Returns the Dependabot ecosystems the repo uses, judged by files at its root."""
    return [name for name, files in DETECT if repo.exists(*files)]

def config(ecosystems, interval = "weekly", group = True, prefixes = {}):
    """Returns the text of a dependabot.yml with one update entry per ecosystem.

    Each entry checks `/` on `interval`. With `group`, minor and patch
    updates come as one PR per ecosystem. Commit messages start with the
    ecosystem's entry in `prefixes`, or `ci(deps)` for github-actions and
    `build(deps)` for the rest.
    """
    out = "version: 2\nupdates:\n"
    for name in ecosystems:
        prefix = prefixes.get(name, "ci(deps)" if name == "github-actions" else "build(deps)")
        out += "  - package-ecosystem: %s\n" % name
        out += "    directory: /\n"
        out += "    schedule:\n      interval: %s\n" % interval
        if group:
            out += "    groups:\n      minor-and-patch:\n        update-types: [minor, patch]\n"
        out += "    commit-message:\n      prefix: %s\n" % json.encode(prefix)
    return out

def write_if_missing(repo, interval = "weekly", group = True, prefixes = {}):
    """Writes .github/dependabot.yml for the detected ecosystems unless a config exists.

    Repos with a config, or with no ecosystem detected, are left alone.
    Returns the ecosystems written, or an empty list.
    """
    if find(repo) != None:
        return []
    found = ecosystems(repo)
    if found:
        repo.write(".github/dependabot.yml", config(found, interval = interval, group = group, prefixes = prefixes))
    return found
