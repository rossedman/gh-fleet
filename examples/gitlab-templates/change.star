load("@fleet//gitlab.star", "bump_component", "set_default", "set_include_ref")

def change(repo):
    set_include_ref(repo, env["TEMPLATES"], env["REF"])
    bump_component(repo, env["COMPONENTS"], env["VERSION"])
    set_default(repo, "interruptible", True)
