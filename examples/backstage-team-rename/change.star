load("@fleet//backstage.star", "entities", "set_owner")

def change(repo):
    before = {(e["path"], e["name"]): e["owner"] for e in entities(repo)}
    set_owner(repo, env["TO"], from_owner = env["FROM"])
    moved = [e["name"] for e in entities(repo) if before.get((e["path"], e["name"])) != e["owner"]]
    repo.set_var("entities", ", ".join(moved))
