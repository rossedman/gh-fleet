load("@fleet//helm.star", "charts", "set_app_version", "set_chart_version", "set_image_tag")
load("@fleet//yamlpath.star", "get")

def change(repo):
    changed = set_image_tag(repo, env["IMAGE"], env["TAG"])
    for chart in charts(repo):
        if not [p for p in changed if p.startswith(chart + "/") or chart == "."]:
            continue
        set_app_version(repo, chart, env["TAG"])
        major, minor, patch = get(repo, chart + "/Chart.yaml", "version").split(".")
        set_chart_version(repo, chart, "%s.%s.%d" % (major, minor, int(patch) + 1))
