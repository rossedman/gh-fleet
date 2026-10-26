load("@fleet//lintconfig.star", "hadolint_ignore", "markdownlint_rule", "yamllint_rule")

def change(repo):
    markdownlint_rule(repo, "MD013", False)
    yamllint_rule(repo, "line-length", {"max": 120})
    if repo.glob("**/Dockerfile"):
        hadolint_ignore(repo, ["DL3008", "DL3013"])
