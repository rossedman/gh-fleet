"""Set up CodeQL code scanning: add a workflow, add languages and move the action.

Edits to existing workflows are line-based, so comments and layout stay
as they were.
"""

load("./gha.star", _bump_action = "bump_action", _workflows = "workflows")

# CodeQL's older language names, mapped to the current ones.
_ALIASES = {
    "c": "c-cpp",
    "cpp": "c-cpp",
    "java": "java-kotlin",
    "javascript": "javascript-typescript",
    "kotlin": "java-kotlin",
    "typescript": "javascript-typescript",
}

# Languages that need a build to analyze; the rest use build-mode none.
_AUTOBUILD = ["c-cpp", "go", "swift"]

_DEFAULT_VERSIONS = {
    "actions/checkout": "v7",
    "github/codeql-action": "v4",
}

def _norm(language):
    language = language.strip().strip("\"'").lower()
    return _ALIASES.get(language, language)

def _build_mode(language):
    return "autobuild" if _norm(language) in _AUTOBUILD else "none"

def _indent(line):
    return len(line) - len(line.lstrip(" "))

def workflows(repo):
    """Lists the workflows that run CodeQL.

    Returns:
        The sorted paths of workflows that use any `github/codeql-action`
        action.
    Example:
        before .github/workflows/codeql.yml:
            steps:
              - uses: github/codeql-action/init@v3
        before .github/workflows/ci.yml:
            steps:
              - uses: actions/checkout@v4
        call:
            workflows(repo)
        returns:
            [".github/workflows/codeql.yml"]
    """
    return [p for p in _workflows(repo) if re.search(r"""uses:\s*["']?github/codeql-action/""", repo.read(p))]

def _languages_in(content):
    out = []
    lines = content.split("\n")
    for i, line in enumerate(lines):
        g = re.groups(r"^\s*(?:-\s+)?language:\s*\[([^\]]*)\]", line)
        if g:
            out += [x for x in g[0].split(",") if x.strip()]
            continue
        g = re.groups(r"^\s*-\s+language:\s*([^\s#]+)", line)
        if g:
            out.append(g[0])
            continue
        g = re.groups(r"^\s*languages:\s*([^#$]+?)\s*(#.*)?$", line)
        if g and "${{" not in line:
            out += [x for x in g[0].split(",") if x.strip()]
            continue
        if re.matches(r"^\s*language:\s*(#.*)?$", line):
            for j in range(i + 1, len(lines)):
                item = re.groups(r"^\s*-\s+([^\s#]+)", lines[j])
                if not item:
                    break
                out.append(item[0])
    return out

def languages(repo):
    """Lists the languages CodeQL analyzes, across every CodeQL workflow.

    Reads `language` matrix lists (flow or block), matrix `include` entries,
    and literal `languages` inputs. Older names are reported by their
    current ones, so `javascript` is `javascript-typescript`.

    Returns:
        The sorted, distinct language names.
    Example:
        before .github/workflows/codeql.yml:
            jobs:
              analyze:
                strategy:
                  matrix:
                    language: ["go", "javascript"]
                steps:
                  - uses: github/codeql-action/init@v3
                    with:
                      languages: ${{ matrix.language }}
        call:
            languages(repo)
        returns:
            ["go", "javascript-typescript"]
    """
    out = []
    for path in workflows(repo):
        for language in _languages_in(repo.read(path)):
            language = _norm(language)
            if language not in out:
                out.append(language)
    return sorted(out)

def _add_to(content, language, build_mode):
    lines = content.split("\n")
    want = _norm(language)
    if want in [_norm(x) for x in _languages_in(content)]:
        return None
    for i, line in enumerate(lines):
        g = re.groups(r"^(\s*(?:-\s+)?language:\s*\[)([^\]]*)(\].*)$", line)
        if g:
            items = [x.strip() for x in g[1].split(",") if x.strip()]
            quote = items[0][0] if items and items[0][0] in "\"'" else ""
            pad = " " if g[1].startswith(" ") else ""
            items.append(quote + language + quote)
            lines[i] = g[0] + pad + ", ".join(items) + pad + g[2]
            return "\n".join(lines)
        if re.matches(r"^\s*language:\s*(#.*)?$", line):
            at = i
            for j in range(i + 1, len(lines)):
                if not re.matches(r"^\s*-\s+\S", lines[j]):
                    break
                at = j
            if at == i:
                continue
            lines.insert(at + 1, lines[at][:_indent(lines[at])] + "- " + language)
            return "\n".join(lines)
    entries = [i for i, line in enumerate(lines) if re.matches(r"^\s*-\s+language:", line)]
    if entries:
        last = entries[-1]
        col = _indent(lines[last])
        end = last
        for j in range(last + 1, len(lines)):
            if not lines[j].strip() or _indent(lines[j]) <= col:
                break
            end = j
        new = [lines[last][:col] + "- language: " + language]
        if "build-mode:" in "\n".join(lines[last:end + 1]):
            new.append(" " * (col + 2) + "build-mode: " + build_mode)
        return "\n".join(lines[:end + 1] + new + lines[end + 1:])
    for i, line in enumerate(lines):
        g = re.groups(r"^(\s*languages:\s*)([^#]+?)(\s*(?:#.*)?)$", line)
        if g and "${{" not in line:
            lines[i] = g[0] + g[1] + ", " + language + g[2]
            return "\n".join(lines)
    return None

def add_language(repo, language, build_mode = None):
    """Adds a language to every CodeQL workflow that doesn't analyze it yet.

    Appends to a `language` matrix list (keeping its quotes), to the
    matrix `include` entries (with a `build-mode` when the others have one),
    or to a literal `languages` input. Older names count as their current
    ones, so a workflow with `javascript` already has
    `javascript-typescript`.

    Args:
        language: the CodeQL language, such as `python` or `actions`.
        build_mode: the `build-mode` for a new `include` entry. Defaults to
            `autobuild` for c-cpp, go and swift, and `none` otherwise.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/codeql.yml:
            jobs:
              analyze:
                strategy:
                  matrix:
                    include:
                      - language: go
                        build-mode: autobuild
                steps:
                  - uses: github/codeql-action/init@v4
        call:
            add_language(repo, "actions")
        after .github/workflows/codeql.yml:
            jobs:
              analyze:
                strategy:
                  matrix:
                    include:
                      - language: go
                        build-mode: autobuild
                      - language: actions
                        build-mode: none
                steps:
                  - uses: github/codeql-action/init@v4
        returns:
            [".github/workflows/codeql.yml"]
    """
    changed = []
    for path in workflows(repo):
        out = _add_to(repo.read(path), language, build_mode or _build_mode(language))
        if out != None:
            repo.write(path, out)
            changed.append(path)
    return changed

def set_version(repo, to, from_ref = None):
    """Moves every `github/codeql-action` action (`init`, `analyze`, `upload-sarif`...) to a new version.

    Refs pinned to a commit SHA are left alone.

    Args:
        to: the new ref, such as `v4`.
        from_ref: when set, only this exact ref is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/codeql.yml:
            steps:
              - uses: github/codeql-action/init@v3
              - uses: github/codeql-action/analyze@v3
        call:
            set_version(repo, "v4")
        after .github/workflows/codeql.yml:
            steps:
              - uses: github/codeql-action/init@v4
              - uses: github/codeql-action/analyze@v4
        returns:
            [".github/workflows/codeql.yml"]
    """
    return _bump_action(repo, "github/codeql-action", to, from_ref = from_ref)

def add_workflow(repo, languages, branch = None, cron = "30 1 * * 1", path = ".github/workflows/codeql.yml", versions = {}):
    """Writes a CodeQL workflow, unless the repo already runs CodeQL.

    The workflow runs on pushes and pull requests to `branch` and weekly,
    with one matrix entry per language and least-privilege permissions.

    Args:
        languages: the CodeQL languages, such as `["go", "actions"]`. Each
            gets `build-mode: autobuild` for c-cpp, go and swift, and
            `none` otherwise.
        branch: the branch to scan; defaults to the repo's default branch,
            or `main` when it isn't known.
        cron: the weekly schedule.
        path: where to write the workflow.
        versions: refs to use instead of the defaults, keyed by action:
            `actions/checkout` (v7) and `github/codeql-action` (v4).
    Returns:
        True when the workflow was written; False when a workflow already
        uses CodeQL.
    Example:
        call:
            add_workflow(repo, ["go", "actions"], branch = "main")
        after .github/workflows/codeql.yml:
            name: CodeQL

            on:
              push:
                branches: [main]
              pull_request:
                branches: [main]
              schedule:
                - cron: "30 1 * * 1"

            permissions:
              contents: read

            jobs:
              analyze:
                name: Analyze (${{ matrix.language }})
                runs-on: ubuntu-latest
                permissions:
                  actions: read
                  contents: read
                  security-events: write
                strategy:
                  fail-fast: false
                  matrix:
                    include:
                      - language: go
                        build-mode: autobuild
                      - language: actions
                        build-mode: none
                steps:
                  - uses: actions/checkout@v7
                  - uses: github/codeql-action/init@v4
                    with:
                      languages: ${{ matrix.language }}
                      build-mode: ${{ matrix.build-mode }}
                  - uses: github/codeql-action/analyze@v4
                    with:
                      category: /language:${{ matrix.language }}
        returns:
            True
    """
    if workflows(repo):
        return False
    if not languages:
        fail("codeql.add_workflow: no languages given")
    v = dict(_DEFAULT_VERSIONS)
    v.update(versions)
    branch = branch or repo.default_branch or "main"
    matrix = "".join([
        "          - language: %s\n            build-mode: %s\n" % (language, _build_mode(language))
        for language in languages
    ])
    repo.write(path, """name: CodeQL

on:
  push:
    branches: [%s]
  pull_request:
    branches: [%s]
  schedule:
    - cron: "%s"

permissions:
  contents: read

jobs:
  analyze:
    name: Analyze (${{ matrix.language }})
    runs-on: ubuntu-latest
    permissions:
      actions: read
      contents: read
      security-events: write
    strategy:
      fail-fast: false
      matrix:
        include:
%s    steps:
      - uses: actions/checkout@%s
      - uses: github/codeql-action/init@%s
        with:
          languages: ${{ matrix.language }}
          build-mode: ${{ matrix.build-mode }}
      - uses: github/codeql-action/analyze@%s
        with:
          category: /language:${{ matrix.language }}
""" % (branch, branch, cron, matrix, v["actions/checkout"], v["github/codeql-action"], v["github/codeql-action"]))
    return True
