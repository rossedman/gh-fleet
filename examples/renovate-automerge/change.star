load("@fleet//renovate.star", "add_package_rule", "replace_preset")

def change(repo):
    replace_preset(repo, "config:base", "config:recommended")
    add_package_rule(repo, {
        "matchDepTypes": ["devDependencies"],
        "matchUpdateTypes": ["minor", "patch"],
        "automerge": True,
    })
