"""Edit Flux HelmReleases and sources: chart versions and Git refs.

Edits are line-based, so comments and layout stay as they were.
"""

load("./yamlpath.star", _set_sibling = "set_sibling")

def _kind(repo, kind):
    out = []
    for path in repo.glob("**/*.yaml", "**/*.yml", exclude = ["**/node_modules/**"]):
        content = repo.read(path)
        if re.matches(r"(?m)^apiVersion:\s*\S*toolkit\.fluxcd\.io/", content) and re.matches(r"(?m)^kind:\s*" + kind + r"\s*$", content):
            out.append(path)
    return out

def helm_releases(repo):
    """Lists the YAML files that define Flux HelmReleases.

    Returns:
        The sorted paths of YAML files with a `helm.toolkit.fluxcd.io`
        apiVersion and `kind: HelmRelease`.
    Example:
        before clusters/prod/redis.yaml:
            apiVersion: helm.toolkit.fluxcd.io/v2
            kind: HelmRelease
        before clusters/prod/ns.yaml:
            apiVersion: v1
            kind: Namespace
        call:
            helm_releases(repo)
        returns:
            ["clusters/prod/redis.yaml"]
    """
    return _kind(repo, "HelmRelease")

def set_chart_version(repo, chart, version):
    """Sets the chart `version` of every HelmRelease that installs a chart.

    Args:
        chart: the chart name, as in `spec.chart.spec.chart`.
        version: the new version or range, such as `19.x`.
    Returns:
        The list of paths that changed.
    Example:
        before clusters/prod/redis.yaml:
            apiVersion: helm.toolkit.fluxcd.io/v2
            kind: HelmRelease
            spec:
              chart:
                spec:
                  chart: redis
                  version: "18.x"
                  sourceRef:
                    kind: HelmRepository
                    name: bitnami
        call:
            set_chart_version(repo, "redis", "19.x")
        after clusters/prod/redis.yaml:
            apiVersion: helm.toolkit.fluxcd.io/v2
            kind: HelmRelease
            spec:
              chart:
                spec:
                  chart: redis
                  version: "19.x"
                  sourceRef:
                    kind: HelmRepository
                    name: bitnami
        returns:
            ["clusters/prod/redis.yaml"]
    """
    return [p for p in helm_releases(repo) if _set_sibling(repo, p, "chart", chart, "version", version)]

def set_git_tag(repo, url, tag):
    """Sets `ref.tag` of every GitRepository that points at a URL.

    Only sources already pinned by `tag` change; sources following a
    branch, semver range or commit are left alone.

    Args:
        url: the repository URL, as in `spec.url`.
        tag: the new tag.
    Returns:
        The list of paths that changed.
    Example:
        before clusters/prod/source.yaml:
            apiVersion: source.toolkit.fluxcd.io/v1
            kind: GitRepository
            spec:
              url: https://github.com/acme/deploy
              ref:
                tag: v1.4.0
        call:
            set_git_tag(repo, "https://github.com/acme/deploy", "v1.5.0")
        after clusters/prod/source.yaml:
            apiVersion: source.toolkit.fluxcd.io/v1
            kind: GitRepository
            spec:
              url: https://github.com/acme/deploy
              ref:
                tag: v1.5.0
        returns:
            ["clusters/prod/source.yaml"]
    """
    changed = []
    for path in _kind(repo, "GitRepository"):
        content = repo.read(path)
        lines = content.split("\n")
        found = False
        for i, line in enumerate(lines):
            g = re.groups(r"""^(\s*)url:\s*["']?([^"'\s#]+)""", line)
            if g:
                found = g[1] == url
                continue
            t = re.groups(r"""^(\s*tag:\s*["']?)([^"'\s#]+)(.*)$""", line)
            if found and t and t[1] != tag:
                lines[i] = t[0] + tag + t[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
