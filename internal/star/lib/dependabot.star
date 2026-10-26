"""Detect package ecosystems, write a Dependabot configuration, and edit existing ones."""

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

def _entries(lines):
    # The `updates` entries: (first line, last line, key column, ecosystem, directory).
    out = []
    for i, line in enumerate(lines):
        g = re.groups(r"""^(\s*)-\s+package-ecosystem:\s*["']?([\w-]+)""", line)
        if not g:
            continue
        indent = len(g[0])
        end = i
        directory = None
        for j in range(i + 1, len(lines)):
            s = lines[j]
            if not s.strip() or s.strip().startswith("#"):
                continue
            if len(s) - len(s.lstrip(" ")) <= indent:
                break
            end = j
            d = re.groups(r"""^\s*directory:\s*["']?([^"'\s#]+)""", s)
            if d and directory == None:
                directory = d[0]
        out.append((i, end, indent + 2, g[1], directory))
    return out

def add_update(repo, ecosystem, directory = "/", interval = "weekly", prefix = None):
    """Adds an update entry to an existing Dependabot config, unless one covers that ecosystem and directory.

    Args:
        ecosystem: the package ecosystem, such as `docker` or `gomod`.
        directory: the directory Dependabot looks in.
        interval: how often to check.
        prefix: the commit message prefix; `ci(deps)` for github-actions
            and `build(deps)` otherwise when None.
    Returns:
        True when the file changed; False when the entry exists or the repo
        has no config (use `write_if_missing` to create one).
    Example:
        before .github/dependabot.yml:
            version: 2
            updates:
              - package-ecosystem: gomod
                directory: /
                schedule:
                  interval: weekly
        call:
            add_update(repo, "docker")
        after .github/dependabot.yml:
            version: 2
            updates:
              - package-ecosystem: gomod
                directory: /
                schedule:
                  interval: weekly
              - package-ecosystem: docker
                directory: /
                schedule:
                  interval: weekly
                commit-message:
                  prefix: "build(deps)"
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    content = repo.read(path)
    lines = content.split("\n")
    entries = _entries(lines)
    for _, _, _, eco, d in entries:
        if eco == ecosystem and (d or "/") == directory:
            return False
    if prefix == None:
        prefix = "ci(deps)" if ecosystem == "github-actions" else "build(deps)"
    dash = (entries[-1][2] - 2) if entries else 2
    pad = " " * dash
    new = [
        pad + "- package-ecosystem: " + ecosystem,
        pad + "  directory: " + directory,
        pad + "  schedule:",
        pad + "    interval: " + interval,
        pad + "  commit-message:",
        pad + "    prefix: " + json.encode(prefix),
    ]
    at = entries[-1][1] + 1 if entries else len(lines)
    if not entries:
        while at > 0 and lines[at - 1].strip() == "":
            at -= 1
        if not re.matches(r"(?m)^updates:", content):
            new = ["updates:"] + new
    lines = lines[:at] + new + lines[at:]
    repo.write(path, "\n".join(lines))
    return True

def set_option(repo, key, value, ecosystem = None):
    """Sets an option on update entries, such as `open-pull-requests-limit` or `rebase-strategy`.

    An existing option is replaced; a missing one is added to the entry.
    Use `schedule.interval` for the check interval.

    Args:
        key: the option, dotted for nested ones, such as `schedule.interval`.
        value: a string, number or boolean.
        ecosystem: only entries for this ecosystem change; every entry when
            None.
    Returns:
        True when the file changed.
    Example:
        before .github/dependabot.yml:
            version: 2
            updates:
              - package-ecosystem: npm
                directory: /
                schedule:
                  interval: daily  # too noisy
              - package-ecosystem: github-actions
                directory: /
                schedule:
                  interval: weekly
        call:
            set_option(repo, "schedule.interval", "weekly", ecosystem = "npm")
            set_option(repo, "open-pull-requests-limit", 5, ecosystem = "npm")
        after .github/dependabot.yml:
            version: 2
            updates:
              - package-ecosystem: npm
                directory: /
                schedule:
                  interval: weekly  # too noisy
                open-pull-requests-limit: 5
              - package-ecosystem: github-actions
                directory: /
                schedule:
                  interval: weekly
        returns:
            True
    """
    path = find(repo)
    if path == None:
        return False
    content = repo.read(path)
    lines = content.split("\n")
    parts = key.split(".")
    encoded = _scalar(value)
    for start, end, col, eco, _ in reversed(_entries(lines)):
        if ecosystem != None and eco != ecosystem:
            continue
        # Find each part of the key at increasing depth within the entry.
        at, depth, lo = -1, col, start
        found = True
        for n, part in enumerate(parts):
            hit = -1
            for j in range(lo, end + 1):
                s = lines[j]
                lead = len(s) - len(s.lstrip(" -"))
                if lead == depth and re.matches(r"^\s*-?\s*" + re.escape(part) + r"\s*:", s):
                    hit = j
                    break
            if hit < 0:
                found = False
                break
            at, lo, depth = hit, hit + 1, depth + 2
        if found:
            g = re.groups(r"""^(\s*-?\s*[\w.-]+:\s*)(["']?[^"'#]*?["']?)(\s*#.*)?$""", lines[at])
            if g and g[1].strip("\"'") != encoded.strip("\""):
                lines[at] = g[0] + encoded + (g[2] or "")
        elif len(parts) == 1:
            lines = lines[:end + 1] + [" " * col + key + ": " + encoded] + lines[end + 1:]
        else:
            fail("dependabot.set_option: %s has no %s to set %s in" % (eco, ".".join(parts[:-1]), parts[-1]))
    updated = "\n".join(lines)
    if updated == content:
        return False
    repo.write(path, updated)
    return True

def _scalar(v):
    if type(v) == "bool":
        return "true" if v else "false"
    if type(v) == "string" and not re.matches(r"^[\w./-]+$", v):
        return json.encode(v)
    return str(v)
