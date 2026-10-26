load("@fleet//swift.star", "set_package", "set_tools_version")

def change(repo):
    if set_package(repo, env["URL"], env["VERSION"]):
        set_tools_version(repo, "5.10")
