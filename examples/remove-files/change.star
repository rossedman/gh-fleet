def change(repo):
    repo.delete(*env["REMOVE"].split())
