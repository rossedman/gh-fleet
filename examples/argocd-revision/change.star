load("@fleet//argocd.star", "set_target_revision")

def change(repo):
    set_target_revision(repo, env["REPO_URL"], env["REVISION"])
