"""Bump chart versions of Helmfile releases.

Edits are line-based, so comments and templating in helmfile.yaml stay as
they were.
"""

load("./yamlpath.star", _set_sibling = "set_sibling")

def files(repo):
    """Lists the repo's Helmfiles.

    Returns:
        The sorted paths of helmfile.yaml, helmfile.yaml.gotmpl and files
        under helmfile.d.
    Example:
        before helmfile.yaml:
            releases: []
        before helmfile.d/10-ingress.yaml:
            releases: []
        call:
            files(repo)
        returns:
            ["helmfile.d/10-ingress.yaml", "helmfile.yaml"]
    """
    return repo.glob("**/helmfile.yaml", "**/helmfile.yaml.gotmpl", "**/helmfile.yml", "**/helmfile.d/*.yaml", "**/helmfile.d/*.yaml.gotmpl")

def set_chart_version(repo, chart, version):
    """Sets the `version` of every release that installs a chart.

    Args:
        chart: the chart as releases write it, such as `bitnami/redis`.
        version: the new version or range.
    Returns:
        The list of paths that changed.
    Example:
        before helmfile.yaml:
            releases:
              - name: cache
                namespace: data
                chart: bitnami/redis
                version: 18.6.1
              - name: ingress
                chart: ingress-nginx/ingress-nginx
                version: 4.9.0
        call:
            set_chart_version(repo, "bitnami/redis", "20.2.1")
        after helmfile.yaml:
            releases:
              - name: cache
                namespace: data
                chart: bitnami/redis
                version: 20.2.1
              - name: ingress
                chart: ingress-nginx/ingress-nginx
                version: 4.9.0
        returns:
            ["helmfile.yaml"]
    """
    return [p for p in files(repo) if _set_sibling(repo, p, "chart", chart, "version", version)]
