"""Bump plugin versions in Packer templates.

Edits are line-based, so comments and alignment in .pkr.hcl files stay as
they were.
"""

def _lit(s):
    return s.replace("$", "$$")

def templates(repo):
    """Lists the repo's Packer HCL templates.

    Returns:
        The sorted paths of every *.pkr.hcl file.
    Example:
        before images/base.pkr.hcl:
            packer {}
        before main.tf:
            terraform {}
        call:
            templates(repo)
        returns:
            ["images/base.pkr.hcl"]
    """
    return repo.glob("**/*.pkr.hcl")

def set_plugin_version(repo, source, constraint):
    """Sets the `version` of a plugin in `required_plugins`, matched by its `source`.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        source: the plugin source, such as `github.com/hashicorp/amazon`.
        constraint: the new constraint, such as `>= 1.3.3`.
    Returns:
        The list of paths that changed.
    Example:
        before images/base.pkr.hcl:
            packer {
              required_plugins {
                amazon = {
                  version = ">= 1.2.0"
                  source  = "github.com/hashicorp/amazon"
                }
              }
            }
        call:
            set_plugin_version(repo, "github.com/hashicorp/amazon", ">= 1.3.3")
        after images/base.pkr.hcl:
            packer {
              required_plugins {
                amazon = {
                  version = ">= 1.3.3"
                  source  = "github.com/hashicorp/amazon"
                }
              }
            }
        returns:
            ["images/base.pkr.hcl"]
    """
    src = r"""\bsource\s*=\s*\"""" + re.escape(source) + '"'
    version = r"""(\bversion\s*=\s*")([^"]*)(")"""
    changed = []
    for path in templates(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            if not re.matches(src, line):
                continue
            start, end = i, i
            if not ("{" in line and "}" in line):
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
