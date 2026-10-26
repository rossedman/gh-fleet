load("@fleet//ruby.star", "set_gem")

def change(repo):
    repo.set_var("changed", len(set_gem(repo, "rails", env["RAILS"])) > 0)
