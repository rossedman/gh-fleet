load("@fleet//cmake.star", "set_fetchcontent_tag", "set_minimum_version")

def change(repo):
    set_minimum_version(repo, env["CMAKE"])
    set_fetchcontent_tag(repo, "googletest", env["GTEST"])
