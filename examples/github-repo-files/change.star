load("@fleet//github.star", "add_contact_link", "add_release_category", "set_label")

def change(repo):
    add_release_category(repo, "Security", ["security"])
    add_release_category(repo, "Dependencies", ["dependencies"])
    add_release_category(repo, "Other changes", ["*"])
    set_label(repo, "fleet", "1f6feb", description = "Opened by gh fleet")
    add_contact_link(
        repo,
        "Report a vulnerability",
        "https://github.com/%s/security/advisories/new" % repo.full_name,
        "Please report security issues privately.",
    )
