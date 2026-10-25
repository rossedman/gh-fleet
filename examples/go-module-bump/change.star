load("@fleet//gomod.star", "outdated")

def change(repo):
    repo.set_var("dirs", outdated(repo, env["MODULE"], env["VERSION"]))
