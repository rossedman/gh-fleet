load("@fleet//circleci.star", "bump_orb", "orbs")

def change(repo):
    old = [o.split("@")[1] for o in orbs(repo) if o.split("@")[0] == env["ORB"]]
    repo.set_var("previous", ", ".join(old) or "none")
    bump_orb(repo, env["ORB"], env["VERSION"])
