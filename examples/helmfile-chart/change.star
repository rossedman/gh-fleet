load("@fleet//helmfile.star", "set_chart_version")

def change(repo):
    set_chart_version(repo, env["CHART"], env["VERSION"])
