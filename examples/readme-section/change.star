load("@fleet//markdown.star", "ensure_badge", "set_section")

SECURITY = """Please report vulnerabilities privately to security@acme.dev, or through
[GitHub's private reporting](https://github.com/%s/security/advisories/new).
Do not open a public issue."""

def change(repo):
    set_section(repo, "README.md", "Security", SECURITY % repo.full_name)
    url = "https://github.com/%s/actions/workflows/ci.yml" % repo.full_name
    ensure_badge(repo, "README.md", "[![CI](%s/badge.svg)](%s)" % (url, url))
