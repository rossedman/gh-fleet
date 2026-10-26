"""Edit GitLab CI configuration: job and service images, included template refs, CI/CD component versions and defaults.

Edits are line-based, so comments, anchors and `!reference` tags stay as
they were.
"""

load("./refs.star", _bump = "bump", _find = "find")
load("./yamlpath.star", _set_in = "set_in", _set_sibling = "set_sibling")

CONFIGS = [".gitlab-ci.yml", ".gitlab-ci.yaml", ".gitlab/**/*.yml", ".gitlab/**/*.yaml", "**/*.gitlab-ci.yml"]

def _lit(s):
    return s.replace("$", "$$")

def configs(repo):
    """Lists the repo's GitLab CI files.

    Returns:
        The sorted paths of .gitlab-ci.yml, of YAML files under .gitlab
        (often included locally), and of any *.gitlab-ci.yml.
    Example:
        before .gitlab-ci.yml:
            stages: [test]
        before .gitlab/ci/test.yml:
            test: {}
        before ci/other.yml:
            x: 1
        call:
            configs(repo)
        returns:
            [".gitlab-ci.yml", ".gitlab/ci/test.yml"]
    """
    return repo.glob(*CONFIGS)

def bump_image(repo, image, from_tag, to_tag):
    """Moves a job or service image from one tag to another.

    Covers `image: node:18`, `image: { name: node:18 }` written as a block,
    and `services` entries written as `- postgres:15` or `- name: postgres:15`.
    A variant suffix is kept, so `node:18-alpine` becomes `node:22-alpine`,
    while `node:18.2` is not touched.

    Args:
        image: the image name as written, including any registry.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before .gitlab-ci.yml:
            default:
              image: node:18-alpine
            test:
              services:
                - postgres:15
                - name: redis:7
              script: npm test
        call:
            bump_image(repo, "node", "18", "22")
            bump_image(repo, "postgres", "15", "17")
        after .gitlab-ci.yml:
            default:
              image: node:22-alpine
            test:
              services:
                - postgres:17
                - name: redis:7
              script: npm test
        returns:
            [".gitlab-ci.yml"]
    """
    pattern = r"""(?m)^(\s*(?:-\s*)?(?:(?:image|name):\s*)?["']?)""" + re.escape(image) + ":" + re.escape(from_tag) + r"""([-@"'\s]|$)"""
    repl = "${1}" + _lit(image) + ":" + _lit(to_tag) + "${2}"
    changed = []
    for path in configs(repo):
        content = repo.read(path)
        updated = re.sub(pattern, repl, content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_include_ref(repo, project, ref):
    """Sets the `ref` of every `include` that pulls templates from another project.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        project: the project path, as in `include: project:`.
        ref: the new branch, tag or commit.
    Returns:
        The list of paths that changed.
    Example:
        before .gitlab-ci.yml:
            include:
              - project: acme/ci-templates
                ref: v1.4.0  # templates
                file: /go.yml
              - local: .gitlab/ci/test.yml
        call:
            set_include_ref(repo, "acme/ci-templates", "v2.0.0")
        after .gitlab-ci.yml:
            include:
              - project: acme/ci-templates
                ref: v2.0.0  # templates
                file: /go.yml
              - local: .gitlab/ci/test.yml
        returns:
            [".gitlab-ci.yml"]
    """
    return [p for p in configs(repo) if _set_sibling(repo, p, "project", project, "ref", ref, never_lower = True)]

def components(repo):
    """Lists the CI/CD components a config includes, with their versions.

    Returns:
        A list of `component@version` strings, in file order.
    Example:
        before .gitlab-ci.yml:
            include:
              - component: gitlab.com/acme/components/go-build@1.2.0
              - component: $CI_SERVER_FQDN/acme/components/sast@~latest
        call:
            components(repo)
        returns:
            ["gitlab.com/acme/components/go-build@1.2.0", "$CI_SERVER_FQDN/acme/components/sast@~latest"]
    """
    out = []
    for path in configs(repo):
        for line in repo.read(path).split("\n"):
            g = re.groups(r"""component:\s*["']?(\S+?)@([^"'\s#]+)""", line)
            if g:
                out.append(g[0] + "@" + g[1])
    return out

def bump_component(repo, component, version, from_ref = None):
    """Moves an included CI/CD component to a new version.

    Args:
        component: the component path before `@`, such as
            `gitlab.com/acme/components/go-build`; `*` matches any run of
            path characters.
        version: the new version, such as `1.3.0` or `~latest`.
        from_ref: when set, only this exact version is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before .gitlab-ci.yml:
            include:
              - component: gitlab.com/acme/components/go-build@1.2.0
                inputs:
                  go_version: "1.22"
        call:
            bump_component(repo, "gitlab.com/acme/components/go-build", "1.3.0")
        after .gitlab-ci.yml:
            include:
              - component: gitlab.com/acme/components/go-build@1.3.0
                inputs:
                  go_version: "1.22"
        returns:
            [".gitlab-ci.yml"]
    """
    return _bump(repo, CONFIGS, component, version, sep = "@", from_ref = from_ref)

def set_default(repo, key, value):
    """Sets a key under `default:` in .gitlab-ci.yml, adding `default:` if needed.

    Use it for settings every job should inherit, such as `interruptible`,
    `timeout` or `retry`. In a config that starts with a `spec:` header
    for its inputs, the key goes in the pipeline after the `---`, not the
    header.

    Args:
        key: the key under `default`, such as `interruptible`; dotted for
            nested keys, such as `retry.max`.
        value: a string, number, boolean or list.
    Returns:
        True when the file changed.
    Example:
        before .gitlab-ci.yml:
            stages: [test]
            default:
              image: node:22
            test:
              script: npm test
        call:
            set_default(repo, "interruptible", True)
        after .gitlab-ci.yml:
            stages: [test]
            default:
              image: node:22
              interruptible: true
            test:
              script: npm test
        returns:
            True
    """
    path = ".gitlab-ci.yml" if repo.exists(".gitlab-ci.yml") else ".gitlab-ci.yaml"
    if not repo.exists(path):
        return False
    content = repo.read(path)
    lines = content.split("\n")
    head = 0
    for i, line in enumerate(lines):
        if re.matches(r"^---\s*$", line):
            if [l for l in lines[:i] if re.matches(r"^spec:", l)]:
                head = i + 1
            break
    body = "\n".join(lines[head:])
    updated = ("\n".join(lines[:head]) + "\n" if head else "") + _set_in(body, "default." + key, value)
    if updated == content:
        return False
    repo.write(path, updated)
    return True
