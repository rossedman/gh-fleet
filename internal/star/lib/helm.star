"""Bump Helm chart versions, chart dependencies and image tags in values files.

Edits are line-based, so comments and layout in Chart.yaml and values
files stay as they were.
"""

load("./yamlpath.star", _set = "set")

def _lit(s):
    return s.replace("$", "$$")

def charts(repo):
    """Lists the repo's Helm charts.

    Returns:
        The sorted directories that hold a Chart.yaml; the repo root is `.`.
    Example:
        before charts/api/Chart.yaml:
            name: api
        before charts/worker/Chart.yaml:
            name: worker
        call:
            charts(repo)
        returns:
            ["charts/api", "charts/worker"]
    """
    return [p.removesuffix("Chart.yaml").rstrip("/") or "." for p in repo.glob("**/Chart.yaml")]

def set_chart_version(repo, chart, version):
    """Sets a chart's `version` in its Chart.yaml.

    Args:
        chart: the chart directory, as `charts` lists it.
        version: the new chart version.
    Returns:
        True when the file changed.
    Example:
        before charts/api/Chart.yaml:
            apiVersion: v2
            name: api
            version: 1.4.0
            appVersion: "2.3.1"
        call:
            set_chart_version(repo, "charts/api", "1.5.0")
        after charts/api/Chart.yaml:
            apiVersion: v2
            name: api
            version: 1.5.0
            appVersion: "2.3.1"
        returns:
            True
    """
    return _set(repo, chart + "/Chart.yaml", "version", version)

def set_app_version(repo, chart, version):
    """Sets a chart's `appVersion` in its Chart.yaml, keeping its quotes.

    Args:
        chart: the chart directory, as `charts` lists it.
        version: the new application version.
    Returns:
        True when the file changed.
    Example:
        before charts/api/Chart.yaml:
            version: 1.4.0
            appVersion: "2.3.1"
        call:
            set_app_version(repo, "charts/api", "2.4.0")
        after charts/api/Chart.yaml:
            version: 1.4.0
            appVersion: "2.4.0"
        returns:
            True
    """
    return _set(repo, chart + "/Chart.yaml", "appVersion", version)

def set_dependency(repo, name, version):
    """Sets the version of a chart dependency in every Chart.yaml that lists it.

    Args:
        name: the dependency's `name`.
        version: the new version or range, such as `15.5.x`.
    Returns:
        The list of paths that changed. Remember to refresh Chart.lock with
        `helm dependency update` in a `run` step if the repo commits it.
    Example:
        before charts/api/Chart.yaml:
            name: api
            dependencies:
              - name: postgresql
                version: 12.1.0  # bitnami
                repository: https://charts.bitnami.com/bitnami
              - name: redis
                version: 17.0.0
        call:
            set_dependency(repo, "postgresql", "15.5.x")
        after charts/api/Chart.yaml:
            name: api
            dependencies:
              - name: postgresql
                version: 15.5.x  # bitnami
                repository: https://charts.bitnami.com/bitnami
              - name: redis
                version: 17.0.0
        returns:
            ["charts/api/Chart.yaml"]
    """
    changed = []
    for chart in charts(repo):
        path = chart + "/Chart.yaml"
        content = repo.read(path)
        lines = content.split("\n")
        items = _items(lines)
        for item in items:
            if item["name"] == name and item["version"] != None:
                i = item["version"]
                lines[i] = re.sub(r"""^(\s*(?:-\s*)?version:\s*["']?)[^"'\s#]+""", "${1}" + _lit(version), lines[i], count = 1)
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def _items(lines):
    # The entries of the top-level dependencies list: name and the line
    # of their version key.
    out = []
    inside = False
    item = None
    item_indent = -1
    for i, line in enumerate(lines):
        if re.matches(r"^dependencies:\s*(#.*)?$", line):
            inside = True
            continue
        if not inside or not line.strip() or line.strip().startswith("#"):
            continue
        if not line.startswith(" ") and not line.startswith("-"):
            break
        g = re.groups(r"^(\s*)-\s*(.*)$", line)
        if g and (item_indent < 0 or len(g[0]) == item_indent):
            item_indent = len(g[0])
            item = {"name": None, "version": None}
            out.append(item)
            line = g[0] + "  " + g[1]
        if item == None:
            continue
        kv = re.groups(r"""^\s*(name|version):\s*["']?([^"'\s#]+)""", line)
        if kv:
            item[kv[0]] = kv[1] if kv[0] == "name" else i
    return out

def set_image_tag(repo, repository, tag, include = ["**/values*.yaml", "**/values*.yml"]):
    """Sets the `tag` next to `repository: <repository>` in values files.

    Matches the common layout where an image is a mapping with sibling
    `repository` and `tag` keys, at any depth. Quotes around the old tag
    are kept; images with no `tag` key are left alone.

    Args:
        repository: the image repository as written, such as `ghcr.io/acme/api`.
        tag: the new tag.
        include: glob patterns of the values files to edit.
    Returns:
        The list of paths that changed.
    Example:
        before charts/api/values.yaml:
            image:
              repository: ghcr.io/acme/api
              tag: "2.3.1"
              pullPolicy: IfNotPresent
            sidecar:
              image:
                repository: envoyproxy/envoy
                tag: v1.29.0
        call:
            set_image_tag(repo, "ghcr.io/acme/api", "2.4.0")
        after charts/api/values.yaml:
            image:
              repository: ghcr.io/acme/api
              tag: "2.4.0"
              pullPolicy: IfNotPresent
            sidecar:
              image:
                repository: envoyproxy/envoy
                tag: v1.29.0
        returns:
            ["charts/api/values.yaml"]
    """
    repo_line = r"""^(\s*)repository:\s*["']?""" + re.escape(repository) + r"""["']?\s*(#.*)?$"""
    changed = []
    for path in repo.glob(*include):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(repo_line, line)
            if not g:
                continue
            indent = len(g[0])
            # Siblings share the indent; look both ways until the mapping ends.
            for step in (-1, 1):
                j = i + step
                for _ in range(len(lines)):
                    if j < 0 or j >= len(lines):
                        break
                    s = lines[j]
                    lead = len(s) - len(s.lstrip(" "))
                    if s.strip() and lead < indent:
                        break
                    t = re.groups(r"""^(\s*tag:\s*["']?)([^"'\s#]+)(.*)$""", s)
                    if t and lead == indent:
                        if t[1] != tag:
                            lines[j] = t[0] + tag + t[2]
                        break
                    j += step
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
