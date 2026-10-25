load("@fleet//cargo.star", "set_dependency")

def change(repo):
    set_dependency(repo, env["CRATE"], env["VERSION"])
