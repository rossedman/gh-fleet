load("@fleet//gomod.star", "set_go_version")
load("@fleet//make.star", "set_var")

def change(repo):
    set_var(repo, "GO_VERSION", env["GO_VERSION"])
    set_go_version(repo, env["GO_VERSION"])
