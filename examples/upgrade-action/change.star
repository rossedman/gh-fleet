load("@fleet//gha.star", "bump_action")

def change(repo):
    bump_action(repo, env["ACTION"], env["TO"], from_ref = env["FROM"])
