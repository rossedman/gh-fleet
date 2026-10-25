load("@fleet//python.star", "set_requirement")

def change(repo):
    changed = set_requirement(repo, env["PACKAGE"], env["SPEC"])
    repo.set_var("files", ", ".join(changed))
