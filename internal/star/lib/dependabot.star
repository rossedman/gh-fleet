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
    """Finds the repo's Dependabot configuration.

    Returns:
        The path of .github/dependabot.yml or .github/dependabot.yaml, or
        None when the repo has neither.
    Example:
        before .github/dependabot.yaml:
            version: 2
        call:
            find(repo)
        returns:
            ".github/dependabot.yaml"
    """
    for path in [".github/dependabot.yml", ".github/dependabot.yaml"]:
        if repo.exists(path):
            return path
    return None

def ecosystems(repo):
    """Detects the package ecosystems a repo uses, from files at its root.

    Recognizes gomod, npm, pip, cargo, bundler, composer, maven, gradle,
    docker and github-actions.

    Returns:
        The Dependabot ecosystem names, in a fixed order.
    Example:
        before go.mod:
            module acme/api
        before package.json:
            {}
        before .github/workflows/ci.yml:
            on: push
        call:
            ecosystems(repo)
        returns:
            ["gomod", "npm", "github-actions"]
    """
    return [name for name, files in DETECT if repo.exists(*files)]

def config(ecosystems, interval = "weekly", group = True, prefixes = {}):
    """Builds the text of a dependabot.yml with one update entry per ecosystem.

    Args:
        ecosystems: the ecosystem names, such as from `ecosystems(repo)`.
        interval: how often each ecosystem is checked.
        group: when True, minor and patch updates come as one PR per
            ecosystem.
        prefixes: commit message prefixes by ecosystem; the default is
            `ci(deps)` for github-actions and `build(deps)` for the rest.
    Returns:
        The YAML text.
    Example:
        call:
            text = config(["gomod"], interval = "daily", group = False)
            repo.write(".github/dependabot.yml", text)
        after .github/dependabot.yml:
            version: 2
            updates:
              - package-ecosystem: gomod
                directory: /
                schedule:
                  interval: daily
                commit-message:
                  prefix: "build(deps)"
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

    Args:
        interval: passed to `config`.
        group: passed to `config`.
        prefixes: passed to `config`.
    Returns:
        The ecosystems written, or an empty list.
    Example:
        before go.mod:
            module acme/api
        call:
            write_if_missing(repo, group = False)
        after .github/dependabot.yml:
            version: 2
            updates:
              - package-ecosystem: gomod
                directory: /
                schedule:
                  interval: weekly
                commit-message:
                  prefix: "build(deps)"
        returns:
            ["gomod"]
    """
    if find(repo) != None:
        return []
    found = ecosystems(repo)
    if found:
        repo.write(".github/dependabot.yml", config(found, interval = interval, group = group, prefixes = prefixes))
    return found
