load("@fleet//changelog.star", "add_entry")
load("@fleet//npm.star", "set_dependency")

def change(repo):
    if set_dependency(repo, env["PACKAGE"], env["RANGE"]):
        add_entry(repo, "Update %s to %s." % (env["PACKAGE"], env["RANGE"]), section = "Security")
