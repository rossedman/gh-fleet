"""Bump Terraform's required_version and provider version constraints.

Edits are line-based, so comments and alignment in .tf files stay as they
were. .terraform.lock.hcl is not touched; run `terraform init -upgrade`
in CI or a `run` step to refresh it.
"""

def files(repo):
    """Returns the paths of every .tf file outside .terraform directories."""
    return repo.glob("**/*.tf", exclude = ["**/.terraform/**"])

def _lit(s):
    return s.replace("$", "$$")

def set_required_version(repo, constraint):
    """Sets every `required_version = "..."` to `constraint`, such as `>= 1.9`.

    Returns the paths that changed.
    """
    pattern = r"""(?m)^(\s*required_version\s*=\s*")[^"]*(")"""
    changed = []
    for path in files(repo):
        content = repo.read(path)
        updated = re.sub(pattern, "${1}" + _lit(constraint) + "${2}", content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_provider_version(repo, source, constraint):
    """Sets the `version` constraint of the provider whose `source` is `source`, such as `hashicorp/aws`.

    Works on `required_providers` entries written on one line
    (`aws = { source = "hashicorp/aws", version = "~> 5.0" }`) or as a block.
    Entries without a `version` are left alone.

    Returns the paths that changed.
    """
    src = r"""\bsource\s*=\s*"(?:registry\.terraform\.io/)?""" + re.escape(source) + '"'
    version = r"""(\bversion\s*=\s*")([^"]*)(")"""
    changed = []
    for path in files(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            if not re.matches(src, line):
                continue
            if "{" in line and "}" in line:
                start, end = i, i
            else:
                start, end = i, i
                for j in range(i, -1, -1):
                    if "{" in lines[j]:
                        start = j
                        break
                for j in range(i, len(lines)):
                    if "}" in lines[j]:
                        end = j
                        break
            for j in range(start, end + 1):
                g = re.groups(version, lines[j])
                if g and g[1] != constraint:
                    lines[j] = re.sub(version, "${1}" + _lit(constraint) + "${3}", lines[j], count = 1)
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
