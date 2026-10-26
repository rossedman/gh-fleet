"""Edit Bitbucket Pipelines: step images and pipe versions.

Edits are line-based, so comments and anchors in bitbucket-pipelines.yml
stay as they were.
"""

load("./docker.star", _bump_image = "bump_image")
load("./refs.star", _bump = "bump")

FILES = ["bitbucket-pipelines.yml"]

def pipes(repo):
    """Lists the pipes the pipeline uses, with their versions.

    Returns:
        The sorted, de-duplicated `name:version` strings.
    Example:
        before bitbucket-pipelines.yml:
            pipelines:
              default:
                - step:
                    script:
                      - pipe: atlassian/aws-s3-deploy:1.1.0
                      - pipe: atlassian/slack-notify:2.0.0
        call:
            pipes(repo)
        returns:
            ["atlassian/aws-s3-deploy:1.1.0", "atlassian/slack-notify:2.0.0"]
    """
    found = set()
    for line in repo.read("bitbucket-pipelines.yml", default = "").split("\n"):
        g = re.groups(r"""^\s*-?\s*pipe:\s*["']?([\w./-]+):([^"'\s#]+)""", line)
        if g:
            found.add(g[0] + ":" + g[1])
    return sorted(found)

def bump_pipe(repo, pipe, version, from_ref = None):
    """Moves a pipe to a new version.

    Args:
        pipe: the pipe, such as `atlassian/aws-s3-deploy`.
        version: the new version, such as `1.6.1`.
        from_ref: when set, only this exact version is replaced.
    Returns:
        The list of paths that changed.
    Example:
        before bitbucket-pipelines.yml:
            script:
              - pipe: atlassian/aws-s3-deploy:1.1.0
                variables:
                  S3_BUCKET: acme-web
        call:
            bump_pipe(repo, "atlassian/aws-s3-deploy", "1.6.1")
        after bitbucket-pipelines.yml:
            script:
              - pipe: atlassian/aws-s3-deploy:1.6.1
                variables:
                  S3_BUCKET: acme-web
        returns:
            ["bitbucket-pipelines.yml"]
    """
    return _bump(repo, FILES, pipe, version, sep = ":", from_ref = from_ref)

def bump_image(repo, image, from_tag, to_tag):
    """Moves the default or a step's `image` from one tag to another, keeping variants.

    Args:
        image: the image name, such as `node`.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before bitbucket-pipelines.yml:
            image: node:18-bullseye
            pipelines:
              default:
                - step:
                    image: python:3.10
        call:
            bump_image(repo, "node", "18", "22")
        after bitbucket-pipelines.yml:
            image: node:22-bullseye
            pipelines:
              default:
                - step:
                    image: python:3.10
        returns:
            ["bitbucket-pipelines.yml"]
    """
    return _bump_image(repo, image, from_tag, to_tag, include = FILES)
