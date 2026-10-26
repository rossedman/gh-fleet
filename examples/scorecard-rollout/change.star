load("@fleet//scorecard.star", "add_badge", "add_workflow")

def change(repo):
    add_workflow(repo, publish = True)
    add_badge(repo)
