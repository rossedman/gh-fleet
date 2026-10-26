load("@fleet//gitlab.star", "bump_image")

def change(repo):
    bump_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
