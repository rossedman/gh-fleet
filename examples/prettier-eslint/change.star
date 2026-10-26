load("@fleet//eslint.star", "find", "set_rule")
load("@fleet//prettier.star", "set_option")

def change(repo):
    set_option(repo, "printWidth", 100)
    set_option(repo, "trailingComma", "all")
    config = find(repo)
    if config and config.startswith("eslint.config."):
        repo.set_var("eslint", "ESLint uses a flat config (%s); apply the rules there by hand." % config)
        return
    set_rule(repo, "eqeqeq", "error")
    set_rule(repo, "no-console", "error")
    repo.set_var("eslint", "")
