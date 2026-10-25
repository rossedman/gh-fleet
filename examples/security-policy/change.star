load("@fleet//files.star", "write_if_missing")

def change(repo):
    # The template ships with the campaign; campaign.read can't reach anything else.
    policy = campaign.read("SECURITY.md.tmpl").replace("{repo}", repo.full_name)
    write_if_missing(repo, "SECURITY.md", policy, also = [".github/SECURITY.md", "docs/SECURITY.md"])
