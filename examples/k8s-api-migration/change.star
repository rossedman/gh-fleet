load("@fleet//k8s.star", "MIGRATIONS", "migrate_api_versions", "resources")

def change(repo):
    old = ["%s/%s" % (r["kind"], r["name"]) for r in resources(repo) if "%s %s" % (r["api_version"], r["kind"]) in MIGRATIONS]
    repo.set_var("resources", ", ".join(old) or "no resources")
    migrate_api_versions(repo)
