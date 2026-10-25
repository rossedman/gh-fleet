---
title: Starlark changes
---

# Starlark changes

A Starlark change is a small Python-like script that edits files in each repo through a `repo` object. It runs inside gh fleet, with no shell and no network, so it behaves the same on every machine and can only touch the files of the clone it is given. That makes a campaign written this way safe to run even when someone else wrote it.

Use Starlark when the change is an edit to files: parsing JSON or YAML, comparing versions, deciding per file. Use a [shell script](writing-changes.md) when you need a toolchain (`go get`, `npm install`). A campaign can use both.

## A first change

```yaml
change:
  starlark: ./change.star
  env:
    LINE: ".env"
```

```python
# change.star
def change(repo):
    lines = repo.read(".gitignore", default = "").splitlines()
    if env["LINE"] in lines:
        return  # nothing written: the repo is a no-op
    repo.append(".gitignore", env["LINE"] + "\n")
```

The file must define `change(repo)`. gh fleet calls it once per repo, in a fresh clone, with the repo root as the starting point for every path. Whatever files differ afterwards are the change, exactly as with a shell script: no difference means `noop`, and an error fails that repo only.

When `starlark` and `run` are both set, the Starlark change runs first and the shell script second, in the same clone:

```yaml
change:
  starlark: ./bump.star      # edit package.json as data
  run: npm install --package-lock-only --ignore-scripts
```

## Checked before anything is cloned

`gh fleet targets`, `plan` and `apply` compile the script and everything it loads, then run its top level, before resolving a single target. A typo is reported with its line and column:

```console
$ gh fleet plan
   ERROR

  Change.starlark: change.star:3:5: undefined: ensure_lines.
```

A failure while changing one repo names the line too, and the full traceback is in that repo's log:

```console
failed   acme/web change.star:12:9: fail: package.json has no name
```

## The `repo` object

| Method | Does |
| --- | --- |
| `repo.read(path, default = ...)` | the file's content; `default` instead of an error when it doesn't exist |
| `repo.write(path, content)` | replace the file, creating it and its directories; an existing file keeps its mode |
| `repo.append(path, content)` | add to the end of the file, creating it if needed |
| `repo.exists(path, ...)` | whether any of the paths exist |
| `repo.delete(path, ...)` | remove files or directories; missing ones are skipped |
| `repo.glob(pattern, ..., exclude = [...])` | matching files, sorted, with `**` for any depth: `repo.glob("**/package.json", exclude = ["**/node_modules/**"])` |
| `repo.set_var(name, value)` | hand a value to the PR templates as `{{ .Vars.name }}` |

| Attribute | Value |
| --- | --- |
| `repo.owner`, `repo.name`, `repo.full_name` | `acme`, `api`, `acme/api` |
| `repo.default_branch`, `repo.language`, `repo.topics` | from GitHub |
| `repo.properties` | the repo's [custom properties](targeting.md#custom-properties), a dict |
| `repo.vars` | the repo's [variables](../configuration.md#variables), plus any set with `set_var` |

Paths are relative to the repo root and written with `/`. Absolute paths, paths with `..` that leave the repo, anything inside `.git`, and symlinks that point outside the clone are all refused.

Every write and delete is logged with the line that made it, so a surprising diff is easy to trace: `$XDG_STATE_HOME/gh-fleet/<campaign>/logs/<owner>__<repo>.log` has lines like `starlark: write package.json (change.star:18:19)`. `print()` goes to the same log.

## Built-in modules

| Name | Provides |
| --- | --- |
| `json` | `json.decode(s)`, `json.encode(x)`, `json.encode_indent(x, indent = "  ")`, `json.indent(s)`. Encoding keeps a dict's key order, so rewriting `package.json` only changes what you changed. |
| `yaml` | `yaml.decode(s)` and `yaml.encode(x)`, with key order kept. `encode` writes a whole new document: comments are dropped and formatting is normalized, so use it for files you generate, not hand-edited ones. |
| `semver` | `semver.compare(a, b)`, `semver.less(a, b)`, `semver.max(...)`, `semver.valid(v)`, `semver.major(v)`, `semver.major_minor(v)`. A leading `v` is optional. |
| `re` | Go [RE2](https://github.com/google/re2/wiki/Syntax) regular expressions: `re.matches`, `re.search`, `re.groups`, `re.find_all`, `re.split`, and `re.sub(pattern, repl, s, count = -1)` with `$1` or `${name}` in `repl`. |
| `env` | the campaign's `change.env`, a read-only dict |

The language is [Starlark](https://github.com/bazelbuild/starlark/blob/master/spec.md) with `while` loops, sets and top-level `if`/`for` allowed. Recursion isn't, and a run is limited by `change.timeout` and a step budget, so a runaway loop fails that repo instead of hanging the campaign.

## Sharing helpers with `load`

A script can load helpers from files next to it:

```python
load("./lib/text.star", "ensure_line")
```

Loads are relative to the loading file and can't leave the campaign's directory.

To share helpers across campaigns and teams, keep them in a repository and load them by a full commit SHA:

```python
load("github.com/acme/fleet-lib@3f9c2e1b7d4a8e0c6f5b2a9d1e7c3b8a4f6d0e2c:security.star", "harden")
```

The library is fetched once with your own git (so private repos work) into `~/.cache/gh-fleet/starlark/` and reused. Only full 40-character SHAs are accepted, never branches or tags, so the code a campaign runs can't change underneath it. Loads inside the library are relative to it and can't leave its checkout.

## Example

[`examples/node-engines`](https://github.com/rossedman/gh-fleet/tree/main/examples/node-engines) requires Node 22 in every `package.json`, keeps key order, never lowers a newer requirement, and tells the PR what it replaced:

```python
def change(repo):
    want = int(env["NODE"])
    repo.set_var("node", want)
    previous = []
    for path in repo.glob("**/package.json", exclude = ["**/node_modules/**"]):
        pkg = json.decode(repo.read(path))
        engines = pkg.get("engines", {})
        current = engines.get("node", "")
        have = required_major(current)
        if have != None and have >= want:
            continue
        previous.append(current or "unset")
        engines["node"] = ">=%d" % want
        pkg["engines"] = engines
        repo.write(path, json.encode_indent(pkg, indent = "  ") + "\n")
    repo.set_var("previous", ", ".join(previous) or "already set")
```
