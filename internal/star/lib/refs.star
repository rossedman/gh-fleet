"""Find, bump and pin versioned references in any file: `name@ref`, `name#ref` and `source?ref=tag`.

The same shape shows up across tools: GitHub Actions and CircleCI orbs
(`actions/checkout@v4`, `circleci/node@5.1.0`), Jenkins shared libraries
(`@Library('ci-lib@v2')`), Buildkite plugins (`docker-compose#v4.14.0`) and
Terraform module sources (`git::https://github.com/acme/modules.git//vpc?ref=v1.2.0`).
These helpers work on any of them, line by line, so comments and layout
are kept. The tool-specific modules, such as `buildkite` and `terragrunt`,
build on them.

A name may use `*` to match any run of characters other than spaces,
quotes, brackets and separators, as in `acme/*` or
`git::https://github.com/acme/modules.git//*`.
"""

# Characters that end a name or a ref.
_STOP = "\\s\"'(),\\[\\]{}&#@?=;<>|"

def _name_re(name):
    if name == "*":
        return "[^" + _STOP + "]+"
    return ("[^" + _STOP + "]*").join([re.escape(p) for p in name.split("*")])

def _pattern(name, sep):
    # Group 1: what precedes the name; 2: the name; 3: the ref.
    return "(^|[\\s\"'(=:\\[,])(" + _name_re(name) + ")" + re.escape(sep) + "([^" + _STOP + ":]+)"

def _sha(ref):
    return re.matches(r"^[0-9a-f]{40}$", ref)

def _lit(s):
    return s.replace("$", "$$")

def find(repo, files, name, sep = "@", exclude = []):
    """Lists every reference to `name` in the files matching `files`.

    Args:
        files: glob patterns of the files to search, such as
            `[".buildkite/**/*.yml"]`.
        name: the name before the separator; `*` matches any run of name
            characters.
        sep: the separator between name and ref: `@`, `#` or `?ref=`.
        exclude: glob patterns of files to skip.
    Returns:
        A list of dicts with `path`, `line` (counting from 1), `name` and
        `ref`, in file and line order.
    Example:
        before .circleci/config.yml:
            orbs:
              node: circleci/node@5.1.0
              aws-cli: circleci/aws-cli@4.0
        call:
            [r["name"] + " " + r["ref"] for r in find(repo, [".circleci/*.yml"], "circleci/*")]
        returns:
            ["circleci/node 5.1.0", "circleci/aws-cli 4.0"]
    """
    pattern = _pattern(name, sep)
    out = []
    for path in repo.glob(exclude = exclude, *files):
        for n, line in enumerate(repo.read(path).split("\n")):
            for m in re.find_all(pattern, line):
                g = re.groups(pattern, m)
                out.append({"path": path, "line": n + 1, "name": g[1], "ref": g[2]})
    return out

def bump(repo, files, name, to, sep = "@", from_ref = None, exclude = []):
    """Moves every reference to `name` to a new ref.

    References pinned to a full commit SHA are left alone unless
    `from_ref` names that SHA. With `from_ref`, only that exact ref is
    replaced, so `v3` never matches `v3.5.0`.

    Args:
        files: glob patterns of the files to edit.
        name: the name before the separator; `*` matches any run of name
            characters.
        to: the new ref.
        sep: the separator between name and ref: `@`, `#` or `?ref=`.
        from_ref: when set, only this exact ref is replaced.
        exclude: glob patterns of files to skip.
    Returns:
        The list of paths that changed.
    Example:
        before Jenkinsfile:
            @Library('ci-lib@v2.3') _
            pipeline { agent any }
        call:
            bump(repo, ["Jenkinsfile", "**/Jenkinsfile"], "ci-lib", "v3.0")
        after Jenkinsfile:
            @Library('ci-lib@v3.0') _
            pipeline { agent any }
        returns:
            ["Jenkinsfile"]
    """
    pattern = _pattern(name, sep)
    changed = []
    for path in repo.glob(exclude = exclude, *files):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            for m in re.find_all(pattern, line):
                g = re.groups(pattern, m)
                ref = g[2]
                if ref == to or (from_ref != None and ref != from_ref) or (from_ref == None and _sha(ref)):
                    continue
                line = line.replace(m, g[0] + g[1] + sep + to, 1)
            lines[i] = line
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def pin(repo, files, name, shas, sep = "@", comment = "#", exclude = []):
    """Pins references to commit SHAs, keeping the old ref in a comment.

    `name@v4` becomes `name@<sha>`, and ` # v4` is added at the end of the
    line so update tools such as Dependabot still know the version. Refs
    missing from `shas` are left as they are.

    Args:
        files: glob patterns of the files to edit.
        name: the name before the separator; `*` matches any run of name
            characters.
        shas: maps `name<sep>ref` to a full commit SHA; a dict, or a string
            of space-separated `name<sep>ref=sha` pairs.
        sep: the separator between name and ref.
        comment: the comment marker of the file type, such as `#` or `//`;
            None to add no comment.
        exclude: glob patterns of files to skip.
    Returns:
        The list of paths that changed.
    Example:
        before .buildkite/pipeline.yml:
            steps:
              - plugins:
                  - docker-compose#v4.14.0:
                      run: app
        call:
            pin(repo, [".buildkite/*.yml"], "docker-compose", {"docker-compose#v4.14.0": "1111111111111111111111111111111111111111"}, sep = "#")
        after .buildkite/pipeline.yml:
            steps:
              - plugins:
                  - docker-compose#1111111111111111111111111111111111111111: # v4.14.0
                      run: app
        returns:
            [".buildkite/pipeline.yml"]
    """
    if type(shas) == "string":
        pairs = {}
        for item in shas.split():
            key, _, sha = item.rpartition("=")
            pairs[key] = sha
        shas = pairs
    for key, sha in shas.items():
        if not _sha(sha):
            fail("refs.pin: %s resolved to %r, not a commit SHA" % (key, sha))
    pattern = _pattern(name, sep)
    changed = []
    for path in repo.glob(exclude = exclude, *files):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            old = []
            for m in re.find_all(pattern, line):
                g = re.groups(pattern, m)
                key = g[1] + sep + g[2]
                if key in shas:
                    line = line.replace(m, g[0] + g[1] + sep + shas[key], 1)
                    old.append(g[2])
            if old and comment != None:
                line = line.rstrip() + " " + comment + " " + ", ".join(old)
            lines[i] = line
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
