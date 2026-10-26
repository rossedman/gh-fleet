"""Bump Buf remote plugins, module dependencies and the buf.yaml format.

Edits are line-based, so comments in buf.yaml and buf.gen.yaml stay as
they were.
"""

load("./refs.star", _bump = "bump")

def _lit(s):
    return s.replace("$", "$$")

GEN = ["**/buf.gen.yaml", "**/buf.gen.*.yaml"]

def set_plugin_version(repo, plugin, version):
    """Moves a remote plugin to a new version in every buf.gen.yaml.

    Covers `remote:` (v1) and `plugin:` / `remote:` under `plugins` (v2),
    written as `buf.build/<owner>/<plugin>:<version>`.

    Args:
        plugin: the plugin, such as `buf.build/protocolbuffers/go`.
        version: the new version, such as `v1.35.1`.
    Returns:
        The list of paths that changed.
    Example:
        before buf.gen.yaml:
            version: v2
            plugins:
              - remote: buf.build/protocolbuffers/go:v1.31.0
                out: gen/go
              - remote: buf.build/grpc/go:v1.3.0
                out: gen/go
        call:
            set_plugin_version(repo, "buf.build/protocolbuffers/go", "v1.35.1")
        after buf.gen.yaml:
            version: v2
            plugins:
              - remote: buf.build/protocolbuffers/go:v1.35.1
                out: gen/go
              - remote: buf.build/grpc/go:v1.3.0
                out: gen/go
        returns:
            ["buf.gen.yaml"]
    """
    return _bump(repo, GEN, plugin, version, sep = ":")

def set_dep_ref(repo, module, ref):
    """Pins a module dependency in buf.yaml to a label or commit, such as `buf.build/googleapis/googleapis:<commit>`.

    Dependencies listed without a ref get one. buf.lock isn't updated; run
    `buf dep update` (or `buf mod update` for v1) in a `run` step.

    Args:
        module: the module, such as `buf.build/googleapis/googleapis`.
        ref: the label or commit.
    Returns:
        The list of paths that changed.
    Example:
        before buf.yaml:
            version: v2
            deps:
              - buf.build/googleapis/googleapis
              - buf.build/bufbuild/protovalidate:v0.6.0
        call:
            set_dep_ref(repo, "buf.build/googleapis/googleapis", "e7f8d366f5264595bcc4cd4139af9973")
            set_dep_ref(repo, "buf.build/bufbuild/protovalidate", "v0.8.0")
        after buf.yaml:
            version: v2
            deps:
              - buf.build/googleapis/googleapis:e7f8d366f5264595bcc4cd4139af9973
              - buf.build/bufbuild/protovalidate:v0.8.0
        returns:
            ["buf.yaml"]
    """
    pattern = r"""(?m)^(\s*-\s*["']?)""" + re.escape(module) + r"""(?::[^"'\s#]+)?(["']?\s*(?:#.*)?)$"""
    changed = []
    for path in repo.glob("**/buf.yaml", "**/buf.work.yaml"):
        content = repo.read(path)
        updated = re.sub(pattern, "${1}" + _lit(module + ":" + ref) + "${2}", content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
