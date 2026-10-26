"""Move step and plugin images in Drone and Woodpecker CI pipelines.

Both tools write steps with an `image:` key, and plugins are images too,
so one helper covers moving a build image and upgrading a plugin.
"""

load("./docker.star", _bump_image = "bump_image")

FILES = [".drone.yml", ".drone.yaml", ".woodpecker.yml", ".woodpecker.yaml", ".woodpecker/*.yml", ".woodpecker/*.yaml"]

def pipelines(repo):
    """Lists the repo's Drone and Woodpecker pipeline files.

    Returns:
        The sorted paths of .drone.yml, .woodpecker.yml and the files under
        .woodpecker.
    Example:
        before .drone.yml:
            kind: pipeline
        before .woodpecker/test.yml:
            steps: {}
        call:
            pipelines(repo)
        returns:
            [".drone.yml", ".woodpecker/test.yml"]
    """
    return repo.glob(*FILES)

def bump_image(repo, image, from_tag, to_tag):
    """Moves a step or plugin image from one tag to another, keeping variants.

    Args:
        image: the image, such as `golang` or `plugins/docker`.
        from_tag: the tag to replace.
        to_tag: the tag to put in its place.
    Returns:
        The list of paths that changed.
    Example:
        before .drone.yml:
            kind: pipeline
            steps:
              - name: test
                image: golang:1.22-alpine
              - name: publish
                image: plugins/docker:20
        call:
            bump_image(repo, "golang", "1.22", "1.23")
            bump_image(repo, "plugins/docker", "20", "21")
        after .drone.yml:
            kind: pipeline
            steps:
              - name: test
                image: golang:1.23-alpine
              - name: publish
                image: plugins/docker:21
        returns:
            [".drone.yml"]
    """
    return _bump_image(repo, image, from_tag, to_tag, include = FILES)
