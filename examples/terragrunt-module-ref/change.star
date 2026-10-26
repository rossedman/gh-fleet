load("@fleet//terragrunt.star", "set_source_ref", "sources")

def change(repo):
    old = sorted(set([s["ref"] for s in sources(repo) if s["source"].startswith(env["SOURCE"].rstrip("*"))]))
    repo.set_var("previous", ", ".join(old))
    set_source_ref(repo, env["SOURCE"], env["REF"])
