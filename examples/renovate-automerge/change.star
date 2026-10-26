load("@fleet//renovate.star", "add_package_rule", "add_preset", "remove_preset")

def change(repo):
    if remove_preset(repo, "config:base"):
        add_preset(repo, "config:recommended")
    add_package_rule(repo, {
        "matchDepTypes": ["devDependencies"],
        "matchUpdateTypes": ["minor", "patch"],
        "automerge": True,
    })
