"""Edit Buildkite pipelines: plugin versions, agent queues and step defaults.

Edits are line-based, so comments, anchors and layout in pipeline files
stay as they were. Images used through the docker and docker-compose
plugins are `image:` lines, so `docker.bump_image` moves them too.
"""

load("./refs.star", _bump = "bump")

PIPELINES = [".buildkite/**/*.yml", ".buildkite/**/*.yaml", "buildkite.yml", "buildkite.yaml"]

def pipelines(repo):
    """Lists the repo's pipeline files.

    Returns:
        The sorted paths of every YAML file under .buildkite, and of
        buildkite.yml or buildkite.yaml at the root.
    Example:
        before .buildkite/pipeline.yml:
            steps: []
        before .buildkite/deploy/pipeline.yaml:
            steps: []
        before ci/other.yml:
            steps: []
        call:
            pipelines(repo)
        returns:
            [".buildkite/deploy/pipeline.yaml", ".buildkite/pipeline.yml"]
    """
    return repo.glob(*PIPELINES)

def plugins(repo):
    """Lists every plugin a pipeline uses, with its version.

    Returns:
        A sorted, de-duplicated list of `name#ref` strings, such as
        `docker-compose#v4.14.0`.
    Example:
        before .buildkite/pipeline.yml:
            steps:
              - label: test
                plugins:
                  - docker-compose#v4.14.0:
                      run: app
                  - "ecr#v2.7.0":
                      login: true
        call:
            plugins(repo)
        returns:
            ["docker-compose#v4.14.0", "ecr#v2.7.0"]
    """
    found = set()
    for path in pipelines(repo):
        for line in repo.read(path).split("\n"):
            g = re.groups(r"""^\s*-\s*["']?([A-Za-z0-9_./:@-]+)#([^\s"':]+)["']?\s*:?\s*(#.*)?$""", line)
            if g:
                found.add(g[0] + "#" + g[1])
    return sorted(found)

def bump_plugin(repo, plugin, to, from_ref = None):
    """Moves a plugin to a new version in every pipeline.

    The plugin matches as written, such as `docker-compose` or
    `acme/deploy`; a full git URL works too. Plugins pinned to a commit SHA
    are left alone unless `from_ref` names it.

    Args:
        plugin: the plugin name as pipelines write it.
        to: the new version, such as `v5.4.1`.
        from_ref: when set, only this exact version is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before .buildkite/pipeline.yml:
            steps:
              - label: test
                plugins:
                  - docker-compose#v4.14.0:
                      run: app
        call:
            bump_plugin(repo, "docker-compose", "v5.4.1")
        after .buildkite/pipeline.yml:
            steps:
              - label: test
                plugins:
                  - docker-compose#v5.4.1:
                      run: app
        returns:
            [".buildkite/pipeline.yml"]
    """
    return _bump(repo, PIPELINES, plugin, to, sep = "#", from_ref = from_ref)

def queues(repo):
    """Lists the agent queues pipelines target.

    Returns:
        The sorted, de-duplicated queue names.
    Example:
        before .buildkite/pipeline.yml:
            agents:
              queue: default
            steps:
              - command: make deploy
                agents: { queue: "deploy" }
        call:
            queues(repo)
        returns:
            ["default", "deploy"]
    """
    found = set()
    for path in pipelines(repo):
        for line in repo.read(path).split("\n"):
            for pattern in (r"""^\s*queue:\s*["']?([^"'\s#}]+)""", r"""\bqueue:\s*["']?([^"',\s}]+)"""):
                g = re.groups(pattern, line)
                if g:
                    found.add(g[0])
                    break
    return sorted(found)

def set_queue(repo, queue, from_queue = None):
    """Points `agents: queue:` at a new queue in every pipeline.

    Covers block (`queue: x` under `agents:`) and flow (`agents: { queue: x }`)
    forms. Pipelines that set no queue are left alone.

    Args:
        queue: the new queue name.
        from_queue: when set, only this queue is replaced, so steps on other
            queues stay where they are.
    Returns:
        The list of paths that changed.
    Example:
        before .buildkite/pipeline.yml:
            agents:
              queue: default  # shared runners
            steps:
              - command: make test
              - command: make deploy
                agents: { queue: "deploy" }
        call:
            set_queue(repo, "linux-small", from_queue = "default")
        after .buildkite/pipeline.yml:
            agents:
              queue: linux-small  # shared runners
            steps:
              - command: make test
              - command: make deploy
                agents: { queue: "deploy" }
        returns:
            [".buildkite/pipeline.yml"]
    """
    changed = []
    for path in pipelines(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(r"""^(.*\bqueue:\s*["']?)([^"',\s#}]+)(.*)$""", line)
            if g and g[1] != queue and (from_queue == None or g[1] == from_queue):
                lines[i] = g[0] + queue + g[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_step_defaults(repo, values):
    """Adds keys, such as `timeout_in_minutes`, to every command step that lacks them.

    A command step is a list item with a `command` or `commands` key, at
    any depth, so steps inside `group` steps count. Keys a step already
    sets are left as they are, so this never overrides a step's choice.

    Args:
        values: a dict of step keys to scalar values, such as
            `{"timeout_in_minutes": 30}`.
    Returns:
        The list of paths that changed.
    Example:
        before .buildkite/pipeline.yml:
            steps:
              - label: test
                command: make test
              - label: e2e
                command: make e2e
                timeout_in_minutes: 60
              - wait
        call:
            set_step_defaults(repo, {"timeout_in_minutes": 20})
        after .buildkite/pipeline.yml:
            steps:
              - label: test
                command: make test
                timeout_in_minutes: 20
              - label: e2e
                command: make e2e
                timeout_in_minutes: 60
              - wait
        returns:
            [".buildkite/pipeline.yml"]
    """
    changed = []
    for path in pipelines(repo):
        content = repo.read(path)
        lines = content.split("\n")
        done = {}
        # Insert into one step at a time and rescan, so the line numbers of
        # the steps that remain stay right.
        for _ in range(len(lines)):
            todo = None
            for start, end, col in _items(lines):
                if start in done:
                    continue
                done[start] = True
                add = _missing(lines, start, end, col, values)
                if add:
                    todo = (end, add)
                    break
            if todo == None:
                break
            end, add = todo
            lines = lines[:end + 1] + add + lines[end + 1:]
            # Steps after this one moved down; forget them so they are seen again.
            done = {k: True for k in done if k <= end}
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def _missing(lines, start, end, col, values):
    # The lines to add to a command step for the keys it lacks.
    keys = {}
    for j in range(start, end + 1):
        g = re.groups(r"^(\s*)(-\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*:", lines[j])
        if g and len(g[0]) + len(g[1]) == col:
            keys[g[2]] = True
    if "command" not in keys and "commands" not in keys:
        return []
    return [" " * col + k + ": " + _scalar(v) for k, v in values.items() if k not in keys]

def _scalar(v):
    if type(v) == "bool":
        return "true" if v else "false"
    if type(v) == "string":
        return json.encode(v)
    return str(v)

def _items(lines):
    # Every list item that holds a mapping: its first and last line and the
    # column its keys start at.
    out = []
    for i, line in enumerate(lines):
        g = re.groups(r"^(\s*)-\s+([A-Za-z_][A-Za-z0-9_]*)\s*:", line)
        if not g:
            continue
        indent = len(g[0])
        col = len(line) - len(line[indent + 1:].lstrip(" "))
        end = i
        for j in range(i + 1, len(lines)):
            s = lines[j]
            if not s.strip() or s.strip().startswith("#"):
                continue
            if len(s) - len(s.lstrip(" ")) <= indent:
                break
            end = j
        out.append((i, end, col))
    return out
