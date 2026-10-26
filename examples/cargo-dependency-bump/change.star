load("@fleet//cargo.star", "set_dependency")

def change(repo):
    repo.set_var("changed", len(set_dependency(repo, env["CRATE"], env["VERSION"])) > 0)
