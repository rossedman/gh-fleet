load("@fleet//dependabot.star", "add_update", "ecosystems", "set_option")

def change(repo):
    added = []
    for eco in ["docker", "github-actions"]:
        if eco in ecosystems(repo) and add_update(repo, eco):
            added.append(eco)
    repo.set_var("added", ", ".join(added) or "nothing new")
    set_option(repo, "open-pull-requests-limit", 5)
