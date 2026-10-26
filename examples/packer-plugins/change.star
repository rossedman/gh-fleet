load("@fleet//packer.star", "set_plugin_version")

def change(repo):
    set_plugin_version(repo, env["SOURCE"], env["CONSTRAINT"])
