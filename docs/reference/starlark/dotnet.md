---
title: dotnet.star
---

# dotnet.star

.NET: bump NuGet packages, target frameworks and the SDK pinned in global.json.

Edits are line-based, so comments and formatting in project files stay as
they were. Central package management (Directory.Packages.props) is
covered alongside individual projects.

```python
load("@fleet//dotnet.star", "projects", "set_package", "set_target_framework", "sdk_version", "set_sdk_version")
```

| Function | What it does |
| --- | --- |
| [`projects`](#projects) | Lists the project files and shared MSBuild props files. |
| [`set_package`](#set_package) | Sets a NuGet package's version wherever a project already references it. |
| [`set_target_framework`](#set_target_framework) | Sets the target framework of every project, such as `net8.0`. |
| [`sdk_version`](#sdk_version) | Reads the .NET SDK version pinned in global.json. |
| [`set_sdk_version`](#set_sdk_version) | Sets the .NET SDK version in a global.json that already pins one. |

## projects

```python
projects(repo)
```

Lists the project files and shared MSBuild props files.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The sorted paths of every .csproj, .fsproj and .vbproj, and every
Directory.Packages.props and Directory.Build.props, outside bin and
obj directories.

**Example**

```python
projects(repo)
# returns ["Directory.Packages.props", "src/Api/Api.csproj"]
```

=== "Before"

    `src/Api/Api.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk.Web" />
    ```

    `Directory.Packages.props`

    ```text
    <Project />
    ```

    `src/Api/obj/Api.csproj`

    ```text
    <Project />
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dotnet.star#L13)

## set_package

```python
set_package(repo, name, version)
```

Sets a NuGet package's version wherever a project already references it.

Covers `<PackageReference>` and central `<PackageVersion>` entries,
with the version as a `Version="..."` attribute or a `<Version>` child
element. Package names match case-insensitively, as NuGet does.
References without a version (under central package management) are
left alone.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `name` | The package ID, such as `Newtonsoft.Json`. |
| `version` | The new version. |

**Returns:** The list of paths that changed.

**Example**

```python
set_package(repo, "Newtonsoft.Json", "13.0.3")
# returns ["src/Api/Api.csproj"]
```

=== "Before"

    `src/Api/Api.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk.Web">
      <ItemGroup>
        <PackageReference Include="Serilog" Version="3.0.1" />
        <PackageReference Include="newtonsoft.json">
          <Version>13.0.1</Version>
        </PackageReference>
      </ItemGroup>
    </Project>
    ```

=== "After"

    `src/Api/Api.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk.Web">
      <ItemGroup>
        <PackageReference Include="Serilog" Version="3.0.1" />
        <PackageReference Include="newtonsoft.json">
          <Version>13.0.3</Version>
        </PackageReference>
      </ItemGroup>
    </Project>
    ```

**Used in:** [`dotnet-8`](https://github.com/rossedman/gh-fleet/tree/main/examples/dotnet-8)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dotnet.star#L41)

## set_target_framework

```python
set_target_framework(repo, framework, from_framework = None)
```

Sets the target framework of every project, such as `net8.0`.

Changes `<TargetFramework>`. A multi-targeting `<TargetFrameworks>`
list changes only when `from_framework` is given, and then only that
entry is replaced, so `net6.0;net48` becomes `net8.0;net48`.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `framework` | The new target framework moniker. |
| `from_framework` | The moniker to replace; when set, other targets are left alone. |

**Returns:** The list of paths that changed.

**Example**

```python
set_target_framework(repo, "net8.0", from_framework = "net6.0")
# returns ["src/Api/Api.csproj", "src/Lib/Lib.csproj"]
```

=== "Before"

    `src/Api/Api.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk">
      <PropertyGroup>
        <TargetFramework>net6.0</TargetFramework>
      </PropertyGroup>
    </Project>
    ```

    `src/Lib/Lib.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk">
      <PropertyGroup>
        <TargetFrameworks>net6.0;netstandard2.0</TargetFrameworks>
      </PropertyGroup>
    </Project>
    ```

=== "After"

    `src/Api/Api.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk">
      <PropertyGroup>
        <TargetFramework>net8.0</TargetFramework>
      </PropertyGroup>
    </Project>
    ```

    `src/Lib/Lib.csproj`

    ```text
    <Project Sdk="Microsoft.NET.Sdk">
      <PropertyGroup>
        <TargetFrameworks>net8.0;netstandard2.0</TargetFrameworks>
      </PropertyGroup>
    </Project>
    ```

**Used in:** [`dotnet-8`](https://github.com/rossedman/gh-fleet/tree/main/examples/dotnet-8)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dotnet.star#L103)

## sdk_version

```python
sdk_version(repo)
```

Reads the .NET SDK version pinned in global.json.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |

**Returns:** The version, or None when the repo has no global.json or it pins no
SDK.

**Example**

```python
sdk_version(repo)
# returns "6.0.400"
```

=== "Before"

    `global.json`

    ```json
    {"sdk": {"version": "6.0.400", "rollForward": "latestFeature"}}
    ```

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dotnet.star#L171)

## set_sdk_version

```python
set_sdk_version(repo, version)
```

Sets the .NET SDK version in a global.json that already pins one.

**Arguments**

| Name | Description |
| --- | --- |
| `repo` | The repo passed to `change(repo)`. |
| `version` | The SDK version, such as `8.0.400`. |

**Returns:** True when the file changed.

**Example**

```python
set_sdk_version(repo, "8.0.400")
# returns True
```

=== "Before"

    `global.json`

    ```json
    {
      "sdk": {
        "version": "6.0.400",
        "rollForward": "latestFeature"
      }
    }
    ```

=== "After"

    `global.json`

    ```json
    {
      "sdk": {
        "version": "8.0.400",
        "rollForward": "latestFeature"
      }
    }
    ```

**Used in:** [`dotnet-8`](https://github.com/rossedman/gh-fleet/tree/main/examples/dotnet-8)

[Source](https://github.com/rossedman/gh-fleet/blob/main/internal/star/lib/dotnet.star#L187)
