load("@fleet//bazel.star", "set_dep", "set_version")

def change(repo):
    set_version(repo, env["BAZEL"])
    set_dep(repo, "rules_go", env["RULES_GO"])
    set_dep(repo, "gazelle", env["GAZELLE"])
