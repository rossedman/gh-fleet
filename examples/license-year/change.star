load("@fleet//license.star", "detect", "set_year")

def change(repo):
    spdx = detect(repo)
    repo.set_var("license", spdx or "unknown")
    if spdx in ("MIT", "Apache-2.0"):
        set_year(repo, env["YEAR"])
