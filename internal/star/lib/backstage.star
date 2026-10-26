"""Edit Backstage catalog-info.yaml files: owners, annotations, tags and other entity fields.

A catalog file can hold several entities as separate YAML documents, so
every edit can be narrowed to entities of one `kind` or one `name`. Edits
are line-based, through `yamlpath`, so comments and layout survive.
"""

load("./yamlpath.star", _get_in = "get_in", _set_in = "set_in")

def files(repo):
    """Lists the repo's Backstage catalog files.

    Returns:
        The sorted paths of every catalog-info.yaml and catalog-info.yml,
        outside node_modules.
    Example:
        before catalog-info.yaml:
            kind: Component
        before services/billing/catalog-info.yml:
            kind: Component
        call:
            files(repo)
        returns:
            ["catalog-info.yaml", "services/billing/catalog-info.yml"]
    """
    return repo.glob("**/catalog-info.yaml", "**/catalog-info.yml", exclude = ["**/node_modules/**"])

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

def _text(doc):
    return "\n".join(doc).rstrip("\n") + "\n"

def _matches(text, kind, name):
    k = _get_in(text, "kind")
    if k == None:
        return False
    if kind != None and k.lower() != kind.lower():
        return False
    return name == None or _get_in(text, "metadata.name") == name

def _edit(repo, kind, name, fn):
    # Applies fn(text) to each matching entity; fn returns the new text.
    # Returns the paths that changed.
    changed = []
    for path in files(repo):
        content = repo.read(path)
        docs = _docs(content)
        out = []
        for sep, doc in docs:
            text = _text(doc)
            if _matches(text, kind, name):
                new = fn(text)
                if new.rstrip("\n") != text.rstrip("\n"):
                    trailing = 0
                    for line in reversed(doc):
                        if line.strip():
                            break
                        trailing += 1
                    doc = new.rstrip("\n").split("\n") + [""] * trailing
            out.append((sep, doc))
        updated = _join(out)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def entities(repo):
    """Lists the entities in every catalog file.

    Returns:
        One dict per entity, in file order, with `path`, `kind`, `name`,
        `type`, `owner`, `lifecycle` and `system`; a field the entity
        doesn't set is None.
    Example:
        before catalog-info.yaml:
            apiVersion: backstage.io/v1alpha1
            kind: Component
            metadata:
              name: billing-api
            spec:
              type: service
              owner: group:payments
              lifecycle: production
            ---
            apiVersion: backstage.io/v1alpha1
            kind: API
            metadata:
              name: billing
            spec:
              type: openapi
              owner: payments
        call:
            [(e["kind"], e["name"], e["owner"], e["lifecycle"]) for e in entities(repo)]
        returns:
            [("Component", "billing-api", "group:payments", "production"), ("API", "billing", "payments", None)]
    """
    out = []
    for path in files(repo):
        for _, doc in _docs(repo.read(path)):
            text = _text(doc)
            kind = _get_in(text, "kind")
            if kind == None:
                continue
            e = {"path": path, "kind": kind, "name": _get_in(text, "metadata.name")}
            for field in ("type", "owner", "lifecycle", "system"):
                e[field] = _get_in(text, "spec." + field)
            out.append(e)
    return out

def _ref_name(ref):
    # "group:default/payments" and "payments" both name the group payments.
    ref = ref.strip()
    if ":" in ref:
        ref = ref.split(":", 1)[1]
    if ref.startswith("default/"):
        ref = ref[len("default/"):]
    return ref

def set_owner(repo, owner, from_owner = None, kind = None, name = None):
    """Sets `spec.owner`, such as after a team is renamed or a service changes hands.

    Args:
        owner: the new owner, written as given: `payments`,
            `group:payments` or `user:jdoe`.
        from_owner: when set, only entities owned by this group change.
            `payments`, `group:payments` and `group:default/payments` all
            match each other.
        kind: only change entities of this kind, such as `Component`.
        name: only change the entity with this `metadata.name`.
    Returns:
        The list of paths that changed.
    Example:
        before catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
            spec:
              owner: group:default/payments  # since 2023
            ---
            kind: Component
            metadata:
              name: search
            spec:
              owner: discovery
        call:
            set_owner(repo, "group:billing", from_owner = "payments")
        after catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
            spec:
              owner: group:billing  # since 2023
            ---
            kind: Component
            metadata:
              name: search
            spec:
              owner: discovery
        returns:
            ["catalog-info.yaml"]
    """

    def fn(text):
        if from_owner != None:
            have = _get_in(text, "spec.owner")
            if have == None or _ref_name(have) != _ref_name(from_owner):
                return text
        return _set_in(text, "spec.owner", owner)

    return _edit(repo, kind, name, fn)

def set_field(repo, key, value, kind = None, name = None, only_missing = False):
    """Sets any entity field by key path, such as `spec.lifecycle` or `spec.system`.

    Args:
        key: the dotted key path, or a list for keys containing dots.
        value: a string, number, boolean, None or list of those.
        kind: only change entities of this kind.
        name: only change the entity with this `metadata.name`.
        only_missing: leave entities that already set the key alone.
    Returns:
        The list of paths that changed.
    Example:
        before catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
            spec:
              type: service
              lifecycle: experimental
            ---
            kind: API
            metadata:
              name: billing
            spec:
              type: openapi
        call:
            set_field(repo, "spec.lifecycle", "production", kind = "Component")
            set_field(repo, "spec.system", "billing", only_missing = True)
        after catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
            spec:
              type: service
              lifecycle: production
              system: billing
            ---
            kind: API
            metadata:
              name: billing
            spec:
              type: openapi
              system: billing
        returns:
            ["catalog-info.yaml"]
    """

    def fn(text):
        if only_missing and _get_in(text, key) != None:
            return text
        return _set_in(text, key, value)

    return _edit(repo, kind, name, fn)

def set_annotation(repo, key, value, kind = None, name = None, only_missing = False):
    """Sets a `metadata.annotations` entry, such as `backstage.io/techdocs-ref`.

    Args:
        key: the annotation, such as `pagerduty.com/service-id`.
        value: its value.
        kind: only change entities of this kind.
        name: only change the entity with this `metadata.name`.
        only_missing: leave entities that already have the annotation
            alone.
    Returns:
        The list of paths that changed.
    Example:
        before catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
              annotations:
                github.com/project-slug: acme/billing-api
        call:
            set_annotation(repo, "backstage.io/techdocs-ref", "dir:.")
        after catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
              annotations:
                github.com/project-slug: acme/billing-api
                backstage.io/techdocs-ref: dir:.
        returns:
            ["catalog-info.yaml"]
    """
    return set_field(repo, ["metadata", "annotations", key], value, kind = kind, name = name, only_missing = only_missing)

def _add_tag(text, tag):
    lines = text.split("\n")
    meta = -1
    for i, line in enumerate(lines):
        if re.matches(r"^metadata:\s*(#.*)?$", line):
            meta = i
            break
    if meta < 0:
        return _set_in(text, "metadata.tags", [tag])
    for i in range(meta + 1, len(lines)):
        line = lines[i]
        if line.strip() and not line.startswith(" ") and not line.strip().startswith("#"):
            break
        g = re.groups(r"^(\s+tags:\s*\[)([^\]]*)(\].*)$", line)
        if g:
            items = [x.strip() for x in g[1].split(",") if x.strip()]
            if tag in [x.strip("\"'") for x in items]:
                return text
            lines[i] = g[0] + ", ".join(items + [tag]) + g[2]
            return "\n".join(lines)
        if re.matches(r"^\s+tags:\s*(#.*)?$", line):
            col = len(line) - len(line.lstrip(" "))
            at = i
            prefix = " " * (col + 2) + "- "
            for j in range(i + 1, len(lines)):
                item = re.groups(r"^(\s*-\s+)([^#]*?)\s*(#.*)?$", lines[j])
                if not item or len(lines[j]) - len(lines[j].lstrip(" ")) < col:
                    break
                if item[1].strip("\"'") == tag:
                    return text
                at, prefix = j, item[0]
            lines.insert(at + 1, prefix + tag)
            return "\n".join(lines)
    return _set_in(text, "metadata.tags", [tag])

def add_tag(repo, tag, kind = None, name = None):
    """Adds a tag to `metadata.tags`, keeping the list's flow or block style.

    Args:
        tag: the tag, such as `go` or `pci`. Backstage tags are lowercase
            letters, digits, `-`, `+` and `#`.
        kind: only change entities of this kind.
        name: only change the entity with this `metadata.name`.
    Returns:
        The list of paths that changed.
    Example:
        before catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
              tags:
                - java
            ---
            kind: API
            metadata:
              name: billing
              tags: [rest]
            ---
            kind: Resource
            metadata:
              name: billing-db
        call:
            add_tag(repo, "pci")
        after catalog-info.yaml:
            kind: Component
            metadata:
              name: billing-api
              tags:
                - java
                - pci
            ---
            kind: API
            metadata:
              name: billing
              tags: [rest, pci]
            ---
            kind: Resource
            metadata:
              name: billing-db
              tags: [pci]
        returns:
            ["catalog-info.yaml"]
    """
    if not re.matches(r"^[a-z0-9+#]+(-[a-z0-9+#]+)*$", tag):
        fail("backstage.add_tag: %r isn't a valid Backstage tag" % tag)
    return _edit(repo, kind, name, lambda text: _add_tag(text, tag))

def ensure(repo, owner, type = "service", lifecycle = "production", name = None, system = None, description = None, path = "catalog-info.yaml"):
    """Writes a Component catalog file, unless the repo already has one.

    The component gets the `github.com/project-slug` annotation, so
    Backstage's GitHub plugins find the repo.

    Args:
        owner: the owning group or user, such as `group:payments`.
        type: the component type, such as `service`, `website` or
            `library`.
        lifecycle: `experimental`, `production` or `deprecated`.
        name: the component name; defaults to the repo name.
        system: the system the component belongs to, if any.
        description: a one-line description, if any.
        path: where to write the file.
    Returns:
        True when the file was written; False when the repo already has a
        catalog file.
    Example:
        call:
            ensure(repo, "group:payments", system = "billing", description = "Billing API.")
        after catalog-info.yaml:
            apiVersion: backstage.io/v1alpha1
            kind: Component
            metadata:
              name: api
              description: Billing API.
              annotations:
                github.com/project-slug: acme/api
            spec:
              type: service
              lifecycle: production
              owner: group:payments
              system: billing
        returns:
            True
    """
    if files(repo):
        return False
    text = "apiVersion: backstage.io/v1alpha1\nkind: Component\nmetadata:\n"
    text = _set_in(text, "metadata.name", name or repo.name)
    if description:
        text = _set_in(text, "metadata.description", description)
    text = _set_in(text, ["metadata", "annotations", "github.com/project-slug"], repo.full_name)
    for key, value in (("type", type), ("lifecycle", lifecycle), ("owner", owner), ("system", system)):
        if value != None:
            text = _set_in(text, "spec." + key, value)
    repo.write(path, text)
    return True
