"""Find GitHub Actions workflows, bump action versions, set inputs and add permissions.

Edits are line-based rather than a YAML round trip, so comments, quoting
and formatting stay exactly as they were.
"""

def _lit(s):
    # Escapes $ for use in a re.sub replacement.
    return s.replace("$", "$$")

def workflows(repo):
    """Lists the repo's workflow files.

    Returns:
        The sorted paths of every .yml and .yaml file directly under
        .github/workflows.
    Example:
        before .github/workflows/ci.yml:
            on: push
        before .github/workflows/release.yaml:
            on: release
        before docs/ci.yml:
            not a workflow
        call:
            workflows(repo)
        returns:
            [".github/workflows/ci.yml", ".github/workflows/release.yaml"]
    """
    return repo.glob(".github/workflows/*.yml", ".github/workflows/*.yaml")

def action_files(repo):
    """Lists the workflows and every local action's metadata file.

    Local actions are the action.yml or action.yaml files a repo keeps for
    its own workflows, often under .github/actions.

    Returns:
        The sorted paths of the workflows and the action.yml and action.yaml
        files anywhere in the repo.
    Example:
        before .github/workflows/ci.yml:
            on: push
        before .github/actions/setup/action.yml:
            runs:
              using: composite
        call:
            action_files(repo)
        returns:
            [".github/actions/setup/action.yml", ".github/workflows/ci.yml"]
    """
    return sorted(set(workflows(repo) + repo.glob("**/action.yml", "**/action.yaml")))

def bump_action(repo, action, to, from_ref = None):
    """Moves an action to a new version wherever a workflow or local action uses it.

    Steps that use a sub-path of the action (`owner/repo/path`) match too.
    Refs pinned to a full commit SHA are left alone; use `pin` for those.
    Quotes and trailing comments are kept.

    Args:
        action: the action as `owner/repo`, such as `actions/checkout`.
        to: the new ref, such as `v4`.
        from_ref: when set, only this exact ref is replaced, so `v3` never
            matches `v3.5.0`; otherwise every non-SHA ref is.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/ci.yml:
            jobs:
              test:
                steps:
                  - uses: actions/checkout@v3
                  - uses: actions/checkout@v3.5.0
                  - uses: actions/setup-go@v5
        call:
            bump_action(repo, "actions/checkout", "v4", from_ref = "v3")
        after .github/workflows/ci.yml:
            jobs:
              test:
                steps:
                  - uses: actions/checkout@v4
                  - uses: actions/checkout@v3.5.0
                  - uses: actions/setup-go@v5
        returns:
            [".github/workflows/ci.yml"]
    """
    pattern = r"""(uses:\s*["']?)""" + re.escape(action) + r"""((?:/[^@\s"']*)?)@([^\s"'#]+)"""
    changed = []
    for path in action_files(repo):
        content = repo.read(path)
        out = []
        dirty = False
        for line in content.split("\n"):
            g = re.groups(pattern, line)
            if g and g[2] != to and not re.matches(r"^[0-9a-f]{40}$", g[2]) and (from_ref == None or g[2] == from_ref):
                line = re.sub(pattern, "${1}" + _lit(action) + "${2}@" + _lit(to), line, count = 1)
                dirty = True
            out.append(line)
        if dirty:
            repo.write(path, "\n".join(out))
            changed.append(path)
    return changed

def set_input(repo, key, value, match = None):
    r"""Sets every `<key>: <value>` line in the workflows, such as an action input.

    Typical keys are `go-version`, `node-version` and `python-version`.
    Quotes around the old value are kept.

    Args:
        key: the key to set.
        value: the new value.
        match: a regular expression the current value must match to be
            replaced; for example `r"^\d"` leaves `stable` and matrix
            expressions alone.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/ci.yml:
            steps:
              - uses: actions/setup-go@v5
                with:
                  go-version: '1.22'
              - uses: actions/setup-go@v5
                with:
                  go-version: stable
        call:
            set_input(repo, "go-version", "1.27", match = r"^\d")
        after .github/workflows/ci.yml:
            steps:
              - uses: actions/setup-go@v5
                with:
                  go-version: '1.27'
              - uses: actions/setup-go@v5
                with:
                  go-version: stable
        returns:
            [".github/workflows/ci.yml"]
    """
    pattern = r"""^(\s*-?\s*""" + re.escape(key) + r""":\s*["']?)([^"'\s#]+)"""
    changed = []
    for path in workflows(repo):
        content = repo.read(path)
        out = []
        for line in content.split("\n"):
            g = re.groups(pattern, line)
            if g and g[1] != value and (match == None or re.matches(match, g[1])):
                line = re.sub(pattern, "${1}" + _lit(value), line, count = 1)
            out.append(line)
        updated = "\n".join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def ensure_permissions(repo, permissions = {"contents": "read"}):
    """Adds a top-level `permissions` block to each workflow that has none.

    The block goes right above the top-level `jobs:` key. Workflows that
    already set top-level permissions, even broader ones, and files without a
    `jobs:` key are left alone.

    Args:
        permissions: the scopes and levels to write, in order.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/ci.yml:
            on: push

            jobs:
              test:
                runs-on: ubuntu-latest
        call:
            ensure_permissions(repo, {"contents": "read"})
        after .github/workflows/ci.yml:
            on: push

            permissions:
              contents: read

            jobs:
              test:
                runs-on: ubuntu-latest
        returns:
            [".github/workflows/ci.yml"]
    """
    block = "permissions:\n" + "".join(["  %s: %s\n" % (k, v) for k, v in permissions.items()]) + "\n"
    changed = []
    for path in workflows(repo):
        content = repo.read(path)
        if re.matches(r"(?m)^permissions:", content) or not re.matches(r"(?m)^jobs:", content):
            continue
        head, sep, tail = content.partition("\njobs:")
        if content.startswith("jobs:"):
            updated = block + content
        else:
            updated = head + "\n" + block + sep.removeprefix("\n") + tail
        repo.write(path, updated)
        changed.append(path)
    return changed

# A remote action reference: owner/repo, an optional sub-path, then @ref.
_USES = r"""^(\s*-?\s*uses:\s*["']?)([A-Za-z0-9_.-]+/[A-Za-z0-9_.-]+)((?:/[^@\s"']*)?)@([^\s"'#]+)(["']?)(.*)$"""

def unpinned(repo):
    """Lists each remote action used at a tag or branch rather than a commit SHA.

    Sub-paths are dropped (`github/codeql-action/init@v3` is listed as
    `github/codeql-action@v3`), since the ref belongs to the repository.
    Local actions (`./path`) and `docker://` images are not listed. Hand the
    list to something that can resolve refs, such as `gh api` in a `run`
    step, and the results to `pin`.

    Returns:
        The sorted, de-duplicated `owner/repo@ref` strings.
    Example:
        before .github/workflows/ci.yml:
            steps:
              - uses: actions/checkout@v4
              - uses: github/codeql-action/init@v3
              - uses: ./.github/actions/setup
              - uses: actions/setup-go@0aaccfd150d50ccaeb58ebd88d36e91967a5f35b # v5
        call:
            unpinned(repo)
        returns:
            ["actions/checkout@v4", "github/codeql-action@v3"]
    """
    found = set()
    for path in action_files(repo):
        for line in repo.read(path).split("\n"):
            g = re.groups(_USES, line)
            if g and not re.matches(r"^[0-9a-f]{40}$", g[3]):
                found.add(g[1] + "@" + g[3])
    return sorted(found)

def pin(repo, shas):
    """Pins actions to commit SHAs, keeping the old ref as a comment.

    `uses: owner/repo@ref` becomes `uses: owner/repo@<sha> # ref`, so
    Dependabot can still update the pin. Refs missing from `shas` are left as
    they are.

    Args:
        shas: maps `owner/repo@ref`, as `unpinned` lists them, to a full
            commit SHA. A dict works, and so does a string of space-separated
            `owner/repo@ref=sha` pairs, which is easy to hand over from a shell
            step. A value that is not a SHA fails the run.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/ci.yml:
            steps:
              - uses: actions/checkout@v4
        call:
            pin(repo, {"actions/checkout@v4": "08eba0b27e820071cde6df949e0beb9ba4906955"})
        after .github/workflows/ci.yml:
            steps:
              - uses: actions/checkout@08eba0b27e820071cde6df949e0beb9ba4906955 # v4
        returns:
            [".github/workflows/ci.yml"]
    """
    if type(shas) == "string":
        pairs = {}
        for item in shas.split():
            key, _, sha = item.partition("=")
            pairs[key] = sha
        shas = pairs
    for key, sha in shas.items():
        if not re.matches(r"^[0-9a-f]{40}$", sha):
            fail("pin: %s resolved to %r, not a commit SHA" % (key, sha))
    changed = []
    for path in action_files(repo):
        content = repo.read(path)
        out = []
        for line in content.split("\n"):
            g = re.groups(_USES, line)
            if g and g[1] + "@" + g[3] in shas:
                rest = g[5]
                comment = " # " + g[3]
                if re.matches(r"^\s*#", rest):
                    rest = ""  # replaces an old ref comment
                line = g[0] + g[1] + g[2] + "@" + shas[g[1] + "@" + g[3]] + g[4] + comment + rest
            out.append(line)
        updated = "\n".join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
