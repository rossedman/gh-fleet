"""Bump Helm chart versions deployed by Skaffold.

Edits are line-based, so comments and profiles in skaffold.yaml stay as
they were. For the `apiVersion` of the config itself, run `skaffold fix`
in a `run` step: schema upgrades rename fields, which it knows how to do.
"""

load("./yamlpath.star", _set_sibling = "set_sibling")

def files(repo):
    """Lists the repo's Skaffold configs.

    Returns:
        The sorted paths of every skaffold.yaml and skaffold.yml.
    Example:
        before skaffold.yaml:
            apiVersion: skaffold/v4beta11
        call:
            files(repo)
        returns:
            ["skaffold.yaml"]
    """
    return repo.glob("**/skaffold.yaml", "**/skaffold.yml")

def set_helm_release_version(repo, chart, version):
    """Sets the `version` of every Helm release that installs a remote chart.

    Args:
        chart: the release's `remoteChart`, such as `bitnami/postgresql`.
        version: the new chart version.
    Returns:
        The list of paths that changed.
    Example:
        before skaffold.yaml:
            deploy:
              helm:
                releases:
                  - name: db
                    remoteChart: bitnami/postgresql
                    version: 13.2.0
        call:
            set_helm_release_version(repo, "bitnami/postgresql", "16.0.3")
        after skaffold.yaml:
            deploy:
              helm:
                releases:
                  - name: db
                    remoteChart: bitnami/postgresql
                    version: 16.0.3
        returns:
            ["skaffold.yaml"]
    """
    return [p for p in files(repo) if _set_sibling(repo, p, "remoteChart", chart, "version", version)]
