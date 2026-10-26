load("@fleet//backstage.star", "add_tag", "ensure", "set_annotation")

def change(repo):
    team = repo.properties.get("team")
    owner = "group:" + team if team else env["FALLBACK_OWNER"]
    ensure(repo, owner)
    if repo.exists("mkdocs.yml"):
        set_annotation(repo, "backstage.io/techdocs-ref", "dir:.", kind = "Component", only_missing = True)
    if repo.language:
        add_tag(repo, repo.language.lower().replace(" ", "-"), kind = "Component")
    repo.set_var("owner", owner)
