load("@fleet//devcontainer.star", "add_feature", "bump_image", "files")

def change(repo):
    bump_image(repo, env["IMAGE"], env["FROM"], env["TO"])
    for path in files(repo):
        add_feature(repo, "ghcr.io/devcontainers/features/github-cli:1", path = path)
