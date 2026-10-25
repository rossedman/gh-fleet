load("@fleet//docker.star", "bump_base_image", "bump_image")

def change(repo):
    changed = bump_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
    changed += bump_base_image(repo, env["IMAGE"], env["FROM_TAG"], env["TO_TAG"])
    repo.set_var("files", ", ".join(changed))
