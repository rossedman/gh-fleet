"""Bump CocoaPods pod versions and the platform version in Podfiles.

Edits are line-based, so comments and layout stay as they were.
Podfile.lock isn't updated; run `pod update <name>` in a `run` step on a
macOS runner when it is committed.
"""

def _lit(s):
    return s.replace("$", "$$")

def podfiles(repo):
    """Lists the repo's Podfiles.

    Returns:
        The sorted paths of every Podfile outside Pods directories.
    Example:
        before ios/Podfile:
            platform :ios, '13.0'
        before ios/Pods/x/Podfile:
            x
        call:
            podfiles(repo)
        returns:
            ["ios/Podfile"]
    """
    return repo.glob("**/Podfile", exclude = ["**/Pods/**"])

def set_pod(repo, name, requirement):
    """Sets a pod's version requirement wherever a Podfile lists it with one.

    A newer version already there is kept, so a campaign never lowers one.

    Pods without a version, or pinned to a `:git` source, are left alone.
    Subspecs match by their full name, such as `Firebase/Analytics`.

    Args:
        name: the pod name.
        requirement: the requirement, such as `~> 5.9`.
    Returns:
        The list of paths that changed.
    Example:
        before ios/Podfile:
            target 'App' do
              pod 'Alamofire', '~> 5.6'
              pod 'SnapKit'
            end
        call:
            set_pod(repo, "Alamofire", "~> 5.9")
        after ios/Podfile:
            target 'App' do
              pod 'Alamofire', '~> 5.9'
              pod 'SnapKit'
            end
        returns:
            ["ios/Podfile"]
    """
    pattern = r"""(?m)^(\s*pod\s+(["'])""" + re.escape(name) + r"""["']\s*,\s*["'])([^"']+)(["'])"""
    changed = []
    for path in podfiles(repo):
        content = repo.read(path)
        updated = re.sub(pattern, lambda m: m[0] if semver.lowers(m[3], requirement) else m[1] + requirement + m[4], content)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_platform(repo, platform, version):
    """Raises the deployment target in `platform :ios, '13.0'`, never lowering it.

    Args:
        platform: the platform, such as `ios`, `osx` or `tvos`.
        version: the minimum version, such as `15.0`.
    Returns:
        The list of paths that changed.
    Example:
        before ios/Podfile:
            platform :ios, '12.0'
            use_frameworks!
        call:
            set_platform(repo, "ios", "15.0")
        after ios/Podfile:
            platform :ios, '15.0'
            use_frameworks!
        returns:
            ["ios/Podfile"]
    """
    pattern = r"""(?m)^(\s*platform\s+:""" + re.escape(platform) + r"""\s*,\s*["'])([^"']+)(["'])"""
    changed = []
    for path in podfiles(repo):
        content = repo.read(path)
        g = re.groups(pattern, content)
        if not g or not semver.less(g[1], version):
            continue
        repo.write(path, re.sub(pattern, "${1}" + _lit(version) + "${3}", content, count = 1))
        changed.append(path)
    return changed
