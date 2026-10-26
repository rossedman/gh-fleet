"""Bump Terraform's required_version and provider version constraints.

Edits are line-based, so comments and alignment in .tf files stay as they
were. .terraform.lock.hcl is not touched; run `terraform init -upgrade`
in CI or a `run` step to refresh it.
"""

def files(repo):
    """Lists the repo's Terraform files.

    Returns:
        The sorted paths of every .tf file outside .terraform directories.
    Example:
        before main.tf:
            terraform {}
        before modules/vpc/main.tf:
            terraform {}
        before .terraform/modules/x/main.tf:
            terraform {}
        call:
            files(repo)
        returns:
            ["main.tf", "modules/vpc/main.tf"]
    """
    return repo.glob("**/*.tf", exclude = ["**/.terraform/**"])

def _lit(s):
    return s.replace("$", "$$")

def set_required_version(repo, constraint):
    """Sets every `required_version` constraint.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        constraint: the new constraint, such as `>= 1.9`.
    Returns:
        The list of paths that changed.
    Example:
        before versions.tf:
            terraform {
              required_version = ">= 1.5"
            }
        call:
            set_required_version(repo, ">= 1.9")
        after versions.tf:
            terraform {
              required_version = ">= 1.9"
            }
        returns:
            ["versions.tf"]
    """
    pattern = r"""(?m)^(\s*required_version\s*=\s*")([^"]*)(")"""
    changed = []
    for path in files(repo):
        content = repo.read(path)
        updated = re.sub(pattern, lambda m: m[0] if semver.lowers(m[2], constraint) else m[1] + constraint + m[3], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_provider_version(repo, source, constraint):
    """Sets the version constraint of a provider in `required_providers`.

    A newer version already there is kept, so a campaign never lowers one.

    Works on entries written as a block or on one line
    (`aws = { source = "hashicorp/aws", version = "~> 5.0" }`), and on
    sources with the `registry.terraform.io/` prefix. Entries without a
    `version` are left alone.

    Args:
        source: the provider source, such as `hashicorp/aws`.
        constraint: the new constraint, such as `~> 5.0`.
    Returns:
        The list of paths that changed.
    Example:
        before versions.tf:
            terraform {
              required_providers {
                aws = {
                  source  = "hashicorp/aws"
                  version = "~> 4.0" # pinned
                }
              }
            }
        call:
            set_provider_version(repo, "hashicorp/aws", "~> 5.0")
        after versions.tf:
            terraform {
              required_providers {
                aws = {
                  source  = "hashicorp/aws"
                  version = "~> 5.0" # pinned
                }
              }
            }
        returns:
            ["versions.tf"]
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
                if g and g[1] != constraint and not semver.lowers(g[1], constraint):
                    lines[j] = re.sub(version, "${1}" + _lit(constraint) + "${3}", lines[j], count = 1)
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed
