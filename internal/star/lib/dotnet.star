""".NET: bump NuGet packages, target frameworks and the SDK pinned in global.json.

Edits are line-based, so comments and formatting in project files stay as
they were. Central package management (Directory.Packages.props) is
covered alongside individual projects.
"""

load("./jsonfile.star", _get = "get", _set = "set")

def _lit(s):
    return s.replace("$", "$$")

def projects(repo):
    """Lists the project files and shared MSBuild props files.

    Returns:
        The sorted paths of every .csproj, .fsproj and .vbproj, and every
        Directory.Packages.props and Directory.Build.props, outside bin and
        obj directories.
    Example:
        before src/Api/Api.csproj:
            <Project Sdk="Microsoft.NET.Sdk.Web" />
        before Directory.Packages.props:
            <Project />
        before src/Api/obj/Api.csproj:
            <Project />
        call:
            projects(repo)
        returns:
            ["Directory.Packages.props", "src/Api/Api.csproj"]
    """
    return repo.glob(
        "**/*.csproj",
        "**/*.fsproj",
        "**/*.vbproj",
        "**/Directory.Packages.props",
        "**/Directory.Build.props",
        exclude = ["**/bin/**", "**/obj/**"],
    )

def set_package(repo, name, version):
    """Sets a NuGet package's version wherever a project already references it.

    A newer version already there is kept, so a campaign never lowers one.

    Covers `<PackageReference>` and central `<PackageVersion>` entries,
    with the version as a `Version="..."` attribute or a `<Version>` child
    element. Package names match case-insensitively, as NuGet does.
    References without a version (under central package management) are
    left alone.

    Args:
        name: the package ID, such as `Newtonsoft.Json`.
        version: the new version.
    Returns:
        The list of paths that changed.
    Example:
        before src/Api/Api.csproj:
            <Project Sdk="Microsoft.NET.Sdk.Web">
              <ItemGroup>
                <PackageReference Include="Serilog" Version="3.0.1" />
                <PackageReference Include="newtonsoft.json">
                  <Version>13.0.1</Version>
                </PackageReference>
              </ItemGroup>
            </Project>
        call:
            set_package(repo, "Newtonsoft.Json", "13.0.3")
        after src/Api/Api.csproj:
            <Project Sdk="Microsoft.NET.Sdk.Web">
              <ItemGroup>
                <PackageReference Include="Serilog" Version="3.0.1" />
                <PackageReference Include="newtonsoft.json">
                  <Version>13.0.3</Version>
                </PackageReference>
              </ItemGroup>
            </Project>
        returns:
            ["src/Api/Api.csproj"]
    """
    ref = r"""(?i)<(PackageReference|PackageVersion)\b[^>]*\b(?:Include|Update)\s*=\s*"\s*""" + re.escape(name) + r"""\s*\""""
    attr = r"""(\bVersion\s*=\s*")([^"]*)(")"""
    keep = lambda m: m[0] if semver.lowers(m[2], version) else m[1] + version + m[3]
    changed = []
    for path in projects(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            if not re.matches(ref, line):
                continue
            if re.matches(r"\bVersion\s*=", line):
                lines[i] = re.sub(attr, keep, line, count = 1)
            elif not re.matches(r"/>\s*$", line):
                for j in range(i + 1, len(lines)):
                    if re.matches(r"</Package(Reference|Version)>", lines[j]):
                        break
                    if "<Version>" in lines[j]:
                        lines[j] = re.sub(r"(<Version>)([^<]*)(</Version>)", keep, lines[j], count = 1)
                        break
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def set_target_framework(repo, framework, from_framework = None):
    """Sets the target framework of every project, such as `net8.0`.

    Changes `<TargetFramework>`. A multi-targeting `<TargetFrameworks>`
    list changes only when `from_framework` is given, and then only that
    entry is replaced, so `net6.0;net48` becomes `net8.0;net48`.

    Args:
        framework: the new target framework moniker.
        from_framework: the moniker to replace; when set, other targets are
            left alone.
    Returns:
        The list of paths that changed.
    Example:
        before src/Api/Api.csproj:
            <Project Sdk="Microsoft.NET.Sdk">
              <PropertyGroup>
                <TargetFramework>net6.0</TargetFramework>
              </PropertyGroup>
            </Project>
        before src/Lib/Lib.csproj:
            <Project Sdk="Microsoft.NET.Sdk">
              <PropertyGroup>
                <TargetFrameworks>net6.0;netstandard2.0</TargetFrameworks>
              </PropertyGroup>
            </Project>
        call:
            set_target_framework(repo, "net8.0", from_framework = "net6.0")
        after src/Api/Api.csproj:
            <Project Sdk="Microsoft.NET.Sdk">
              <PropertyGroup>
                <TargetFramework>net8.0</TargetFramework>
              </PropertyGroup>
            </Project>
        after src/Lib/Lib.csproj:
            <Project Sdk="Microsoft.NET.Sdk">
              <PropertyGroup>
                <TargetFrameworks>net8.0;netstandard2.0</TargetFrameworks>
              </PropertyGroup>
            </Project>
        returns:
            ["src/Api/Api.csproj", "src/Lib/Lib.csproj"]
    """
    changed = []
    for path in projects(repo):
        content = repo.read(path)
        lines = content.split("\n")
        for i, line in enumerate(lines):
            g = re.groups(r"^(.*<TargetFramework>)\s*([^<]*?)\s*(</TargetFramework>.*)$", line)
            if g and g[1] != framework and (from_framework == None or g[1] == from_framework):
                lines[i] = g[0] + framework + g[2]
                continue
            g = re.groups(r"^(.*<TargetFrameworks>)\s*([^<]*?)\s*(</TargetFrameworks>.*)$", line)
            if g and from_framework != None:
                tfms = [t.strip() for t in g[1].split(";")]
                if from_framework in tfms:
                    out = []
                    for t in tfms:
                        t = framework if t == from_framework else t
                        if t not in out:
                            out.append(t)
                    lines[i] = g[0] + ";".join(out) + g[2]
        updated = "\n".join(lines)
        if updated != content:
            repo.write(path, updated)
            changed.append(path)
    return changed

def sdk_version(repo):
    """Reads the .NET SDK version pinned in global.json.

    Returns:
        The version, or None when the repo has no global.json or it pins no
        SDK.
    Example:
        before global.json:
            {"sdk": {"version": "6.0.400", "rollForward": "latestFeature"}}
        call:
            sdk_version(repo)
        returns:
            "6.0.400"
    """
    return _get(repo, "global.json", "sdk.version")

def set_sdk_version(repo, version):
    """Sets the .NET SDK version in a global.json that already pins one.

    A newer version already there is kept, so a campaign never lowers one.

    Args:
        version: the SDK version, such as `8.0.400`.
    Returns:
        True when the file changed.
    Example:
        before global.json:
            {
              "sdk": {
                "version": "6.0.400",
                "rollForward": "latestFeature"
              }
            }
        call:
            set_sdk_version(repo, "8.0.400")
        after global.json:
            {
              "sdk": {
                "version": "8.0.400",
                "rollForward": "latestFeature"
              }
            }
        returns:
            True
    """
    current = sdk_version(repo)
    if current == None or semver.lowers(current, version):
        return False
    return _set(repo, "global.json", "sdk.version", version)
