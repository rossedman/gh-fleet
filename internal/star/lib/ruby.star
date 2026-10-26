"""Bump gem versions, the Ruby version and Bundler, and set RuboCop's target Ruby.

Edits are line-based, so comments and layout in Gemfiles and gemspecs
stay as they were. Gemfile.lock isn't re-resolved; follow with a `run`
step such as `bundle lock --update <gem>` when the lockfile is committed.
"""

load("./yamlpath.star", _get = "get", _set = "set")

def _lit(s):
    return s.replace("$", "$$")

def gemfiles(repo):
    """Lists the Gemfiles and gemspecs.

    Returns:
        The sorted paths of every Gemfile, gems.rb and *.gemspec outside
        vendor directories.
    Example:
        before Gemfile:
            source "https://rubygems.org"
        before acme.gemspec:
            Gem::Specification.new
        before vendor/bundle/x/Gemfile:
            x
        call:
            gemfiles(repo)
        returns:
            ["Gemfile", "acme.gemspec"]
    """
    return repo.glob("**/Gemfile", "**/gems.rb", "**/*.gemspec", exclude = ["**/vendor/**"])

def set_gem(repo, name, requirement):
    """Sets a gem's version requirement wherever a Gemfile or gemspec lists it.

    A newer version already there is kept, so a campaign never lowers one.

    Covers `gem "x", "~> 1.0"` with any options after it, and gemspec
    `add_dependency`, `add_runtime_dependency` and
    `add_development_dependency`. Gems listed without a version are left
    alone, since that is a deliberate choice to follow the lockfile.

    Args:
        name: the gem name.
        requirement: the requirement, such as `~> 7.2`, or a list for
            several, such as `[">= 1.5", "< 2"]`.
    Returns:
        The list of paths that changed.
    Example:
        before Gemfile:
            gem "rails", "~> 7.0.8"
            gem 'puma', '>= 5', require: false
            gem "pg"
        call:
            set_gem(repo, "rails", "~> 7.2")
            set_gem(repo, "puma", [">= 6", "< 7"])
        after Gemfile:
            gem "rails", "~> 7.2"
            gem 'puma', '>= 6', '< 7', require: false
            gem "pg"
        returns:
            ["Gemfile"]
    """
    reqs = [requirement] if type(requirement) == "string" else list(requirement)
    head = r"""^(\s*(?:gem|\w+\.add_(?:runtime_|development_)?dependency)\s*\(?\s*(["'])""" + re.escape(name) + r"""["'])((?:\s*,\s*["'][^"']*["'])+)(.*)$"""
    changed = []
    for path in gemfiles(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(head, line)
            if not g:
                continue
            q = g[1]
            if semver.lowers(g[2], reqs[0]):
                continue
            new = g[0] + "".join([", " + q + r + q for r in reqs]) + g[3]
            lines[i] = new
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_ruby(repo, version):
    """Sets the Ruby version a Gemfile pins with `ruby "x"`.

    A newer version already there is kept, so a campaign never lowers one.

    `ruby file: ".ruby-version"` lines are left alone; use
    `runtime.set_version(repo, "ruby", ...)` for the version file.

    Args:
        version: the Ruby version or requirement, such as `3.3.5` or `~> 3.3`.
    Returns:
        The list of paths that changed.
    Example:
        before Gemfile:
            source "https://rubygems.org"
            ruby "3.1.4"
        call:
            set_ruby(repo, "3.3.5")
        after Gemfile:
            source "https://rubygems.org"
            ruby "3.3.5"
        returns:
            ["Gemfile"]
    """
    changed = []
    for path in repo.glob("**/Gemfile", "**/gems.rb", exclude = ["**/vendor/**"]):
        content = repo.read(path)
        updated = re.sub(r"""(?m)^(\s*ruby\s*\(?\s*["'])([^"']*)(["'])""", lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version + m[3], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_required_ruby(repo, requirement):
    """Sets `required_ruby_version` in every gemspec that declares it.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        requirement: the requirement, such as `>= 3.1`.
    Returns:
        The list of paths that changed.
    Example:
        before acme.gemspec:
            Gem::Specification.new do |spec|
              spec.name = "acme"
              spec.required_ruby_version = ">= 2.7"
            end
        call:
            set_required_ruby(repo, ">= 3.1")
        after acme.gemspec:
            Gem::Specification.new do |spec|
              spec.name = "acme"
              spec.required_ruby_version = ">= 3.1"
            end
        returns:
            ["acme.gemspec"]
    """
    changed = []
    for path in repo.glob("**/*.gemspec", exclude = ["**/vendor/**"]):
        content = repo.read(path)
        updated = re.sub(r"""(?m)^(\s*\w+\.required_ruby_version\s*=\s*["'])([^"']*)(["'])""", lambda m: m[0] if semver.lowers(m[2], requirement) else m[1] + requirement + m[3], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_bundler(repo, version):
    """Sets the Bundler version a Gemfile.lock was `BUNDLED WITH`.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        version: the Bundler version, such as `2.5.22`.
    Returns:
        The list of paths that changed.
    Example:
        before Gemfile.lock:
            DEPENDENCIES
              rails

            BUNDLED WITH
               2.4.10
        call:
            set_bundler(repo, "2.5.22")
        after Gemfile.lock:
            DEPENDENCIES
              rails

            BUNDLED WITH
               2.5.22
        returns:
            ["Gemfile.lock"]
    """
    changed = []
    for path in repo.glob("**/Gemfile.lock", "**/gems.locked", exclude = ["**/vendor/**"]):
        content = repo.read(path)
        updated = re.sub(r"(BUNDLED WITH\n\s+)(\S+)", lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version, content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_rubocop_target(repo, version):
    """Sets `AllCops: TargetRubyVersion` in .rubocop.yml.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        version: the Ruby version RuboCop should target, such as `3.3`.
    Returns:
        True when the file changed; False when the repo has no .rubocop.yml.
    Example:
        before .rubocop.yml:
            AllCops:
              TargetRubyVersion: 3.0
              NewCops: enable
        call:
            set_rubocop_target(repo, "3.3")
        after .rubocop.yml:
            AllCops:
              TargetRubyVersion: 3.3
              NewCops: enable
        returns:
            True
    """
    if not repo.exists(".rubocop.yml"):
        return False
    current = _get(repo, ".rubocop.yml", "AllCops.TargetRubyVersion")
    if current != None and semver.lowers(str(current), version):
        return False
    value = float(version) if re.matches(r"^\d+\.\d+$", version) else version
    return _set(repo, ".rubocop.yml", "AllCops.TargetRubyVersion", value)
