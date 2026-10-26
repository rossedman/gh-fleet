"""Edit Argo CD Application manifests: move an application to a new revision or chart version.

Edits are line-based, so comments and layout stay as they were.
"""

load("./yamlpath.star", _set_sibling = "set_sibling")

def applications(repo):
    """Lists the YAML files that define Argo CD Applications or ApplicationSets.

    Returns:
        The sorted paths of YAML files with `apiVersion: argoproj.io/...` and
        `kind: Application` or `kind: ApplicationSet`.
    Example:
        before apps/api.yaml:
            apiVersion: argoproj.io/v1alpha1
            kind: Application
        before apps/svc.yaml:
            apiVersion: v1
            kind: Service
        call:
            applications(repo)
        returns:
            ["apps/api.yaml"]
    """
    out = []
    for path in repo.glob("**/*.yaml", "**/*.yml", exclude = ["**/node_modules/**"]):
        content = repo.read(path)
        if re.matches(r"(?m)^apiVersion:\s*argoproj\.io/", content) and re.matches(r"(?m)^kind:\s*Application(Set)?\s*$", content):
            out.append(path)
    return out

def set_target_revision(repo, repo_url, revision):
    """Sets `targetRevision` for every source that points at a repository.

    Works for `source` and each entry of `sources`. A Helm chart from a chart
    repository is matched by its `repoURL` too, and its `targetRevision` is
    the chart version.

    Args:
        repo_url: the source's `repoURL`, as written.
        revision: the new branch, tag, commit or chart version.
    Returns:
        The list of paths that changed.
    Example:
        before apps/api.yaml:
            apiVersion: argoproj.io/v1alpha1
            kind: Application
            spec:
              source:
                repoURL: https://github.com/acme/deploy.git
                path: apps/api
                targetRevision: v1.4.0
        call:
            set_target_revision(repo, "https://github.com/acme/deploy.git", "v1.5.0")
        after apps/api.yaml:
            apiVersion: argoproj.io/v1alpha1
            kind: Application
            spec:
              source:
                repoURL: https://github.com/acme/deploy.git
                path: apps/api
                targetRevision: v1.5.0
        returns:
            ["apps/api.yaml"]
    """
    return [p for p in applications(repo) if _set_sibling(repo, p, "repoURL", repo_url, "targetRevision", revision)]
