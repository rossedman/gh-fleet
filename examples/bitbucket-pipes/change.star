load("@fleet//bitbucket.star", "bump_image", "bump_pipe", "pipes")

def change(repo):
    old = [p.split(":")[1] for p in pipes(repo) if p.split(":")[0] == env["PIPE"]]
    repo.set_var("previous", ", ".join(old) or "none")
    bump_pipe(repo, env["PIPE"], env["VERSION"])
    bump_image(repo, "node", "18", "22")
