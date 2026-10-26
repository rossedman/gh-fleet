"""Raise Android SDK levels and the Android Gradle Plugin version.

Google Play requires a recent `targetSdk` every year, which makes this a
fleet-wide change for anyone with several apps. Edits are line-based and
work on Groovy and Kotlin DSL build files and version catalogs.
"""

load("./gradle.star", _build_files = "build_files", _set_dependency = "set_dependency", _set_plugin = "set_plugin")

_LEVELS = {
    "compile": r"compileSdk(?:Version)?",
    "target": r"targetSdk(?:Version)?",
    "min": r"minSdk(?:Version)?",
}

AGP_PLUGINS = [
    "com.android.application",
    "com.android.library",
    "com.android.test",
    "com.android.dynamic-feature",
]

def sdk_levels(repo):
    """Reads the SDK levels every module's build file sets.

    Returns:
        A dict of build file path to a dict of `compile`, `target` and
        `min` levels (as integers), for files that set any.
    Example:
        before app/build.gradle.kts:
            android {
                compileSdk = 34
                defaultConfig {
                    minSdk = 24
                    targetSdk = 34
                }
            }
        call:
            sdk_levels(repo)["app/build.gradle.kts"]["target"]
        returns:
            34
    """
    out = {}
    for path in _build_files(repo):
        if path.endswith(".toml"):
            continue
        levels = {}
        for kind, key in _LEVELS.items():
            g = re.groups(r"(?m)^\s*" + key + r"\s*(?:=\s*|\(\s*)?(\d+)", repo.read(path))
            if g:
                levels[kind] = int(g[0])
        if levels:
            out[path] = levels
    return out

def set_sdk(repo, compile = None, target = None, min = None):
    """Raises the compile, target and min SDK levels, never lowering one.

    Covers `compileSdk 34`, `compileSdkVersion 33`, `targetSdk = 34` and
    `minSdk(24)` forms in every module, and `compileSdk`, `targetSdk` and
    `minSdk` entries in a version catalog's `[versions]`.

    Args:
        compile: the compileSdk level to raise to, or None.
        target: the targetSdk level to raise to, or None.
        min: the minSdk level to raise to, or None.
    Returns:
        The list of paths that changed.
    Example:
        before app/build.gradle:
            android {
                compileSdkVersion 33
                defaultConfig {
                    minSdkVersion 21
                    targetSdkVersion 33
                }
            }
        call:
            set_sdk(repo, compile = 35, target = 35)
        after app/build.gradle:
            android {
                compileSdkVersion 35
                defaultConfig {
                    minSdkVersion 21
                    targetSdkVersion 35
                }
            }
        returns:
            ["app/build.gradle"]
    """
    want = {"compile": compile, "target": target, "min": min}
    changed = []
    for path in _build_files(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            for kind, key in _LEVELS.items():
                level = want[kind]
                if level == None:
                    continue
                if path.endswith(".toml"):
                    pattern = r"""^(\s*""" + key + r"""\s*=\s*["']?)(\d+)(.*)$"""
                else:
                    pattern = r"^(\s*" + key + r"\s*(?:=\s*|\(\s*)?)(\d+)(.*)$"
                g = re.groups(pattern, line)
                if g and int(g[1]) < level:
                    lines[i] = g[0] + str(level) + g[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_agp_version(repo, version):
    """Moves the Android Gradle Plugin to a version.

    A newer version already there is kept, so a campaign never lowers one.

    Sets the `com.android.*` plugins wherever they are declared (plugin
    blocks or a version catalog) and the
    `com.android.tools.build:gradle` classpath of older builds. Upgrading AGP
    often needs a newer Gradle wrapper too; see
    `gradle.set_wrapper_version`.

    Args:
        version: the AGP version, such as `8.7.2`.
    Returns:
        The list of paths that changed.
    Example:
        before build.gradle:
            buildscript {
                dependencies {
                    classpath 'com.android.tools.build:gradle:7.4.2'
                }
            }
        call:
            set_agp_version(repo, "8.7.2")
        after build.gradle:
            buildscript {
                dependencies {
                    classpath 'com.android.tools.build:gradle:8.7.2'
                }
            }
        returns:
            ["build.gradle"]
    """
    changed = []
    for plugin in AGP_PLUGINS:
        for p in _set_plugin(repo, plugin, version):
            if p not in changed:
                changed.append(p)
    for p in _set_dependency(repo, "com.android.tools.build:gradle", version):
        if p not in changed:
            changed.append(p)
    return sorted(changed)
