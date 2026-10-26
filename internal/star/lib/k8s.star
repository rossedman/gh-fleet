"""Edit Kubernetes manifests: migrate removed API versions and set labels on every resource.

Works on multi-document YAML files, one document at a time. Edits are
line-based, so comments and layout stay as they were. To move images to a
new tag, use `docker.bump_image`; for Kustomize overlays, `kustomize`.
"""

load("./yamlpath.star", _get_in = "get_in", _set_in = "set_in")

# API versions Kubernetes removed, where only the apiVersion line must
# change. Moves that also change the schema, such as Ingress from
# networking.k8s.io/v1beta1, are left out on purpose.
MIGRATIONS = {
    "policy/v1beta1 PodDisruptionBudget": "policy/v1",
    "batch/v1beta1 CronJob": "batch/v1",
    "autoscaling/v2beta2 HorizontalPodAutoscaler": "autoscaling/v2",
    "storage.k8s.io/v1beta1 CSIStorageCapacity": "storage.k8s.io/v1",
    "flowcontrol.apiserver.k8s.io/v1beta3 FlowSchema": "flowcontrol.apiserver.k8s.io/v1",
    "flowcontrol.apiserver.k8s.io/v1beta3 PriorityLevelConfiguration": "flowcontrol.apiserver.k8s.io/v1",
}

# Where each workload kind keeps its pod template's labels.
_TEMPLATES = {
    "Deployment": ["spec", "template", "metadata", "labels"],
    "StatefulSet": ["spec", "template", "metadata", "labels"],
    "DaemonSet": ["spec", "template", "metadata", "labels"],
    "ReplicaSet": ["spec", "template", "metadata", "labels"],
    "Job": ["spec", "template", "metadata", "labels"],
    "CronJob": ["spec", "jobTemplate", "spec", "template", "metadata", "labels"],
}

def _docs(content):
    # Splits a file into documents: a list of (separator line or None, lines).
    out = []
    sep = None
    current = []
    for line in content.split("\n"):
        if re.matches(r"^---(\s|$)", line):
            out.append((sep, current))
            sep, current = line, []
        else:
            current.append(line)
    out.append((sep, current))
    return out

def _join(docs):
    lines = []
    for sep, doc in docs:
        if sep != None:
            lines.append(sep)
        lines.extend(doc)
    return "\n".join(lines)

def _edit(doc, fn):
    # Applies fn to one document's text, keeping its trailing blank lines.
    text = "\n".join(doc)
    trailing = 0
    for line in reversed(doc):
        if line.strip():
            break
        trailing += 1
    new = fn(text.rstrip("\n") + "\n")
    if new.rstrip("\n") == text.rstrip("\n"):
        return doc
    return new.rstrip("\n").split("\n") + [""] * trailing

def manifests(repo):
    """Lists the YAML files that hold Kubernetes resources.

    Returns:
        The sorted paths of YAML files with at least one document that has
        top-level `apiVersion` and `kind` keys, outside node_modules and
        Helm chart templates.
    Example:
        before k8s/api.yaml:
            apiVersion: apps/v1
            kind: Deployment
        before .github/workflows/ci.yml:
            on: push
        call:
            manifests(repo)
        returns:
            ["k8s/api.yaml"]
    """
    out = []
    for path in repo.glob("**/*.yaml", "**/*.yml", exclude = ["**/node_modules/**", "**/templates/**", ".github/**"]):
        content = repo.read(path)
        if re.matches(r"(?m)^apiVersion:", content) and re.matches(r"(?m)^kind:", content):
            out.append(path)
    return out

def resources(repo):
    """Lists every resource in the repo's manifests.

    Returns:
        A list of dicts with `path`, `api_version`, `kind` and `name`, in
        file and document order.
    Example:
        before k8s/api.yaml:
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: api
            ---
            apiVersion: v1
            kind: Service
            metadata:
              name: api
        call:
            ["%s %s/%s" % (r["api_version"], r["kind"], r["name"]) for r in resources(repo)]
        returns:
            ["apps/v1 Deployment/api", "v1 Service/api"]
    """
    out = []
    for path in manifests(repo):
        for _, doc in _docs(repo.read(path)):
            text = "\n".join(doc)
            kind = _get_in(text, "kind")
            if kind == None:
                continue
            out.append({"path": path, "api_version": _get_in(text, "apiVersion"), "kind": kind, "name": _get_in(text, "metadata.name")})
    return out

def migrate_api_versions(repo, migrations = MIGRATIONS):
    """Moves resources off API versions Kubernetes removed.

    The defaults cover moves where only `apiVersion` changes, such as
    `policy/v1beta1` PodDisruptionBudgets to `policy/v1` and `batch/v1beta1`
    CronJobs to `batch/v1`.

    Args:
        migrations: maps `"<apiVersion> <Kind>"` to the new apiVersion.
    Returns:
        The list of paths that changed.
    Example:
        before k8s/api.yaml:
            apiVersion: policy/v1beta1  # old
            kind: PodDisruptionBudget
            metadata:
              name: api
            ---
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: api
        call:
            migrate_api_versions(repo)
        after k8s/api.yaml:
            apiVersion: policy/v1  # old
            kind: PodDisruptionBudget
            metadata:
              name: api
            ---
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: api
        returns:
            ["k8s/api.yaml"]
    """
    changed = []
    for path in manifests(repo):
        content = repo.read(path)
        docs = []
        for sep, doc in _docs(content):
            text = "\n".join(doc)
            key = "%s %s" % (_get_in(text, "apiVersion"), _get_in(text, "kind"))
            if key in migrations:
                doc = _edit(doc, lambda t: _set_in(t, "apiVersion", migrations[key]))
            docs.append((sep, doc))
        updated = _join(docs)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_label(repo, key, value, kinds = None, pod_templates = True):
    """Sets a label on every resource, and on the pods workloads create.

    The label goes in `metadata.labels`, added if missing. With
    `pod_templates`, Deployments, StatefulSets, DaemonSets, ReplicaSets,
    Jobs and CronJobs get it in their pod template too, so the pods carry
    it. Selectors are never touched.

    Args:
        key: the label key, such as `app.kubernetes.io/part-of`.
        value: the label value.
        kinds: the kinds to label, such as `["Deployment", "Service"]`;
            every kind when None.
        pod_templates: whether to label workloads' pod templates too.
    Returns:
        The list of paths that changed.
    Example:
        before k8s/api.yaml:
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: api
            spec:
              template:
                spec:
                  containers: []
        call:
            set_label(repo, "team", "payments")
        after k8s/api.yaml:
            apiVersion: apps/v1
            kind: Deployment
            metadata:
              name: api
              labels:
                team: payments
            spec:
              template:
                spec:
                  containers: []
                metadata:
                  labels:
                    team: payments
        returns:
            ["k8s/api.yaml"]
    """
    changed = []
    for path in manifests(repo):
        content = repo.read(path)
        docs = []
        for sep, doc in _docs(content):
            kind = _get_in("\n".join(doc), "kind")
            if kind != None and (kinds == None or kind in kinds):
                doc = _edit(doc, lambda t: _set_in(t, ["metadata", "labels", key], value))
                if pod_templates and kind in _TEMPLATES:
                    doc = _edit(doc, lambda t: _set_in(t, _TEMPLATES[kind] + [key], value))
            docs.append((sep, doc))
        updated = _join(docs)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
