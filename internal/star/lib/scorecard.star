"""Set up OpenSSF Scorecard: add the workflow, move the action and add the README badge."""

load("./gha.star", _bump_action = "bump_action", _workflows = "workflows")

_DEFAULT_VERSIONS = {
    "actions/checkout": "v7",
    "actions/upload-artifact": "v7",
    "github/codeql-action": "v4",
    "ossf/scorecard-action": "v2.5.0",
}

def workflow(repo):
    """Finds the workflow that runs Scorecard.

    Returns:
        The path of the first workflow that uses `ossf/scorecard-action`,
        or None.
    Example:
        before .github/workflows/scorecards.yml:
            steps:
              - uses: ossf/scorecard-action@v2.3.1
        call:
            workflow(repo)
        returns:
            ".github/workflows/scorecards.yml"
    """
    for path in _workflows(repo):
        if re.search(r"""uses:\s*["']?ossf/scorecard-action@""", repo.read(path)):
            return path
    return None

def set_version(repo, to, from_ref = None):
    """Moves `ossf/scorecard-action` to a new version.

    Refs pinned to a commit SHA are left alone.

    Args:
        to: the new ref, such as `v2.5.0`.
        from_ref: when set, only this exact ref is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before .github/workflows/scorecard.yml:
            steps:
              - uses: ossf/scorecard-action@v2.3.1 # pinned by hand
        call:
            set_version(repo, "v2.5.0")
        after .github/workflows/scorecard.yml:
            steps:
              - uses: ossf/scorecard-action@v2.5.0 # pinned by hand
        returns:
            [".github/workflows/scorecard.yml"]
    """
    return _bump_action(repo, "ossf/scorecard-action", to, from_ref = from_ref)

def add_workflow(repo, branch = None, cron = "30 1 * * 6", publish = True, path = ".github/workflows/scorecard.yml", versions = {}):
    """Writes the Scorecard workflow, unless the repo already runs Scorecard.

    The workflow follows the one the Scorecard project recommends: it runs
    on pushes to `branch`, on branch protection changes and weekly, and
    uploads results to code scanning.

    Args:
        branch: the branch to analyze; defaults to the repo's default
            branch, or `main` when it isn't known.
        cron: the weekly schedule.
        publish: publish results to the public Scorecard API, which the
            badge needs. Private repos should pass False.
        path: where to write the workflow.
        versions: refs to use instead of the defaults, keyed by action:
            `actions/checkout` (v7), `actions/upload-artifact` (v7),
            `github/codeql-action` (v4) and `ossf/scorecard-action`
            (v2.5.0).
    Returns:
        True when the workflow was written; False when one already runs
        Scorecard.
    Example:
        call:
            add_workflow(repo, branch = "main", publish = False)
        after .github/workflows/scorecard.yml:
            name: Scorecard

            on:
              branch_protection_rule:
              push:
                branches: [main]
              schedule:
                - cron: "30 1 * * 6"

            permissions: read-all

            jobs:
              analysis:
                name: Scorecard analysis
                runs-on: ubuntu-latest
                permissions:
                  security-events: write
                steps:
                  - uses: actions/checkout@v7
                    with:
                      persist-credentials: false
                  - uses: ossf/scorecard-action@v2.5.0
                    with:
                      results_file: results.sarif
                      results_format: sarif
                      publish_results: false
                  - uses: actions/upload-artifact@v7
                    with:
                      name: SARIF file
                      path: results.sarif
                      retention-days: 5
                  - uses: github/codeql-action/upload-sarif@v4
                    with:
                      sarif_file: results.sarif
        returns:
            True
    """
    if workflow(repo) != None:
        return False
    v = dict(_DEFAULT_VERSIONS)
    v.update(versions)
    branch = branch or repo.default_branch or "main"
    id_token = "      id-token: write\n" if publish else ""
    repo.write(path, """name: Scorecard

on:
  branch_protection_rule:
  push:
    branches: [%s]
  schedule:
    - cron: "%s"

permissions: read-all

jobs:
  analysis:
    name: Scorecard analysis
    runs-on: ubuntu-latest
    permissions:
      security-events: write
%s    steps:
      - uses: actions/checkout@%s
        with:
          persist-credentials: false
      - uses: ossf/scorecard-action@%s
        with:
          results_file: results.sarif
          results_format: sarif
          publish_results: %s
      - uses: actions/upload-artifact@%s
        with:
          name: SARIF file
          path: results.sarif
          retention-days: 5
      - uses: github/codeql-action/upload-sarif@%s
        with:
          sarif_file: results.sarif
""" % (branch, cron, id_token, v["actions/checkout"], v["ossf/scorecard-action"], "true" if publish else "false", v["actions/upload-artifact"], v["github/codeql-action"]))
    return True

def add_badge(repo, path = "README.md"):
    """Adds the OpenSSF Scorecard badge to the README, below its first heading.

    The badge needs published results (`add_workflow(publish = True)`).
    Badge lines that follow the heading, such as other badges, stay in
    front of it.

    Args:
        path: the README to edit.
    Returns:
        True when the badge was added; False when the README already links
        to the repo's Scorecard results, or doesn't exist.
    Example:
        before README.md:
            # api

            [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)

            The API service.
        call:
            add_badge(repo)
        after README.md:
            # api

            [![CI](https://github.com/acme/api/actions/workflows/ci.yml/badge.svg)](https://github.com/acme/api/actions)
            [![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/acme/api/badge)](https://scorecard.dev/viewer/?uri=github.com/acme/api)

            The API service.
        returns:
            True
    """
    content = repo.read(path, default = None)
    if content == None:
        return False
    project = "github.com/" + repo.full_name
    if "scorecard.dev/projects/" + project in content or "securityscorecards.dev/projects/" + project in content:
        return False
    badge = "[![OpenSSF Scorecard](https://api.scorecard.dev/projects/%s/badge)](https://scorecard.dev/viewer/?uri=%s)" % (project, project)
    lines = content.split("\n")
    head = -1
    for i, line in enumerate(lines):
        if line.startswith("# "):
            head = i
            break
    if head < 0:
        repo.write(path, badge + "\n\n" + content)
        return True
    last = -1
    for j in range(head + 1, len(lines)):
        if not lines[j].strip() and last < 0:
            continue
        if not re.matches(r"^\s*\[?!\[", lines[j]):
            break
        last = j
    if last >= 0:
        lines.insert(last + 1, badge)
    else:
        lines = lines[:head + 1] + ["", badge] + lines[head + 1:]
    repo.write(path, "\n".join(lines))
    return True
