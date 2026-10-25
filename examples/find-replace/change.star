load("@fleet//files.star", "replace_all")

def change(repo):
    changed = replace_all(repo, env["FROM"], env["TO"], exclude = ["**/node_modules/**", "**/vendor/**"])
    repo.set_var("files", len(changed))
