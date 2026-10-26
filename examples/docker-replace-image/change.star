load("@fleet//docker.star", "replace_base_image")

def change(repo):
    changed = replace_base_image(repo, env["FROM_IMAGE"], env["TO_IMAGE"])
    repo.set_var("dockerfiles", ", ".join(changed))
