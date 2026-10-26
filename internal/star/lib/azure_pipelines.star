"""Edit Azure Pipelines YAML: hosted agent images, task versions and template refs.

Edits are line-based, so comments and layout stay as they were.
"""

load("./refs.star", _bump = "bump")
load("./yamlpath.star", _set_sibling = "set_sibling")

FILES = ["azure-pipelines.yml", "azure-pipelines.yaml", "**/azure-pipelines*.yml", "**/azure-pipelines*.yaml", ".azure-pipelines/**/*.yml", ".azure-pipelines/**/*.yaml", ".azuredevops/**/*.yml"]

def pipelines(repo):
    """Lists the repo's Azure Pipelines files.

    Returns:
        The sorted paths of azure-pipelines*.yml files anywhere and of YAML
        files under .azure-pipelines and .azuredevops.
    Example:
        before azure-pipelines.yml:
            trigger: [main]
        before .azure-pipelines/templates/build.yml:
            steps: []
        call:
            pipelines(repo)
        returns:
            [".azure-pipelines/templates/build.yml", "azure-pipelines.yml"]
    """
    return repo.glob(*FILES)

def set_vm_image(repo, image, from_image = None):
    """Sets `vmImage` on every pool that uses a Microsoft-hosted image.

    Block (`pool:` then `vmImage:`) and flow (`pool: { vmImage: ... }`) forms
    both work.

    Args:
        image: the new image, such as `ubuntu-24.04`.
        from_image: when set, only this image is replaced, such as a
            retired `ubuntu-20.04`.
    Returns:
        The list of paths that changed.
    Example:
        before azure-pipelines.yml:
            pool:
              vmImage: ubuntu-20.04  # hosted
            jobs:
              - job: windows
                pool:
                  vmImage: windows-2022
        call:
            set_vm_image(repo, "ubuntu-24.04", from_image = "ubuntu-20.04")
        after azure-pipelines.yml:
            pool:
              vmImage: ubuntu-24.04  # hosted
            jobs:
              - job: windows
                pool:
                  vmImage: windows-2022
        returns:
            ["azure-pipelines.yml"]
    """
    changed = []
    for path in pipelines(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(r"""^(.*\bvmImage:\s*["']?)([^"',\s#}]+)(.*)$""", line)
            if g and g[1] != image and (from_image == None or g[1] == from_image):
                lines[i] = g[0] + image + g[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def bump_task(repo, task, major, from_major = None):
    """Moves a task to a new major version, such as `Docker@1` to `Docker@2`.

    Args:
        task: the task name, such as `PublishPipelineArtifact`.
        major: the new major version, such as `1`.
        from_major: when set, only this version is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before azure-pipelines.yml:
            steps:
              - task: Docker@1
                inputs:
                  command: build
              - task: DockerCompose@0
        call:
            bump_task(repo, "Docker", "2")
        after azure-pipelines.yml:
            steps:
              - task: Docker@2
                inputs:
                  command: build
              - task: DockerCompose@0
        returns:
            ["azure-pipelines.yml"]
    """
    return _bump(repo, FILES, task, str(major), sep = "@", from_ref = None if from_major == None else str(from_major))

def set_template_ref(repo, name, ref):
    """Sets the `ref` of a template repository under `resources.repositories`.

    A newer version already there is kept, so a campaign never lowers one.

    The repository is matched by its `name` (the project/repo path), and a
    missing `ref` is added.

    Args:
        name: the repository's `name`, such as `acme/pipeline-templates`.
        ref: the new ref, such as `refs/tags/v2.0.0`.
    Returns:
        The list of paths that changed.
    Example:
        before azure-pipelines.yml:
            resources:
              repositories:
                - repository: templates
                  type: github
                  name: acme/pipeline-templates
                  ref: refs/tags/v1.4.0
        call:
            set_template_ref(repo, "acme/pipeline-templates", "refs/tags/v2.0.0")
        after azure-pipelines.yml:
            resources:
              repositories:
                - repository: templates
                  type: github
                  name: acme/pipeline-templates
                  ref: refs/tags/v2.0.0
        returns:
            ["azure-pipelines.yml"]
    """
    return [p for p in pipelines(repo) if _set_sibling(repo, p, "name", name, "ref", ref, never_lower = True)]
