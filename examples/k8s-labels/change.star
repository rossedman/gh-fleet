load("@fleet//k8s.star", "set_label")

def change(repo):
    set_label(repo, "app.kubernetes.io/part-of", repo.name)
    set_label(repo, "acme.dev/team", repo.vars["team"])
