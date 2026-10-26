load("@fleet//kustomize.star", "set_image")

def change(repo):
    changed = set_image(repo, env["IMAGE"], new_tag = env["TAG"])
    repo.set_var("overlays", ", ".join([p.removesuffix("/kustomization.yaml") for p in changed]))
