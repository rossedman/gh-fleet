load("@fleet//gha.star", "ensure_permissions")

def change(repo):
    ensure_permissions(repo, {"contents": "read"})
