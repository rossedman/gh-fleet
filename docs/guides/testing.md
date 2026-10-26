# Testing a campaign

`gh fleet test` runs a campaign's change on files on your machine and checks the result. Nothing is cloned and nothing touches GitHub, so it is quick enough to run on every edit and in CI.

There are two ways to use it:

- **Test cases** in a `tests/` directory next to `fleet.yaml`: sample repos with the files you expect the change to produce.
- **`--dir`**: run the change on a copy of a directory you already have, such as a clone, and print the diff.

## Test cases

Each case is a directory under `tests/`:

```text
fleet.yaml
change.star
tests/
  upgrade/
    before/            the repo's files before the change
      .python-version
      pyproject.toml
    after/             the files the change must produce
      .python-version
      pyproject.toml
  already-newer/
    before/
      pyproject.toml   no after/: the change must leave this repo alone
  flat-config/
    before/
      eslint.config.js
    case.yaml          optional: the repo to run as, and what to check
```

For each case, `gh fleet test`:

1. Copies `before/` into a temporary git repository, so steps that call `git` behave as they would in a clone.
2. Runs every step of the change, shell steps included.
3. Checks that the files match `after/` exactly. Without `after/`, it checks that nothing changed.
4. When files changed, renders the PR title and body, which catches template errors.
5. Runs the change again on the result. A second run must change nothing, which is what happens when gh fleet re-runs a campaign on a repo whose PR is already open.

```console
$ gh fleet test
ok   already-newer
FAIL upgrade
     files differ from after/
── upgrade
diff --git want/pyproject.toml got/pyproject.toml
index b55c601..1af1c4b 100644
--- want/pyproject.toml
+++ got/pyproject.toml
@@ -1,3 +1,3 @@
 [project]
 name = "api"
-requires-python = ">=3.11"
+requires-python = ">=3.12"

## step python version
starlark: write .python-version (@fleet//runtime.star:98:23)
starlark: write pyproject.toml (@fleet//python.star:140:15)
starlark: write .github/workflows/ci.yml (@fleet//gha.star:155:23)
1 passed, 1 failed
```

A failing case prints its step output too: each `starlark: write` line names the file and the library line that wrote it. Pass `-v` to see it for every case, along with the PR title.

### case.yaml

Every field is optional.

| Field | Meaning |
| --- | --- |
| `repo` | The `owner/name` the change runs as. Default `acme/test`. |
| `defaultBranch`, `language`, `topics`, `properties` | What `repo.default_branch`, `repo.language`, `repo.topics` and `repo.properties` return, and what `vars.match` rules see. |
| `vars` | Variables set on top of the campaign's `vars` for this repo. |
| `env` | Step environment, on top of `change.env` and each step's `env`. |
| `wantVars` | Variables the steps must set, compared as text. |
| `wantTitle` | The PR title the case must render. |
| `wantError` | Text the change's error must contain. The case then passes only if the change fails. |

```yaml
repo: acme/web
language: TypeScript
vars:
  today: "2026-01-15"
wantVars:
  expired: CVE-2024-1234
wantTitle: "chore: remove expired .trivyignore entries"
```

### Writing the expected files

Write `before/`, then let the change write `after/` and review it:

```console
$ gh fleet test --update
upd  upgrade
upd  already-newer
0 passed, 0 failed, 2 updated
$ git diff --no-index tests/upgrade/before tests/upgrade/after
```

`--update` replaces each case's `after/` with what the change produced, and removes `after/` when the change left the repo alone. Review the result before committing it: from then on, it is what the test expects.

`--run` limits a run to the cases whose names match a regular expression:

```console
$ gh fleet test --run 'newer|flat' -v
```

### Good cases to write

The cases that catch real bugs are the repos that don't look like your example:

- A repo that is **already newer** than the version you are rolling out. It must come out unchanged.
- A repo that **doesn't use** what the campaign changes. A shell step that runs anyway, such as `cargo update --package tokio` in a repo without tokio, fails the run.
- **Formatting** you want kept: one-line JSON lists, comments, tabs, a file without a final newline.
- The odd layouts real repos have: a README that starts with a logo instead of a title, a `.gitlab-ci.yml` with a `spec:` header, ESLint rules with options.

## Running on a local directory

`--dir` runs the change on a copy of a directory and prints the diff, the variables the steps set and the PR title. The directory itself is never modified.

```console
$ gh fleet test --dir ../billing-api --repo acme/billing-api
── acme/billing-api
diff --git a/.python-version b/.python-version
index 2c07333..e4fba21 100644
--- a/.python-version
+++ b/.python-version
@@ -1 +1 @@
-3.11
+3.12
diff --git a/pyproject.toml b/pyproject.toml
index fd70b45..1af1c4b 100644
--- a/pyproject.toml
+++ b/pyproject.toml
@@ -1,3 +1,3 @@
 [project]
 name = "api"
-requires-python = ">=3.10"
+requires-python = ">=3.12"

title build: python 3.12
```

It is a `plan` for one repo you already have on disk. Use it to try a change on real repos before you write cases for them, or to debug a repo that `plan` reported as failed. `--repo` sets the `owner/name` the change runs as; it defaults to `acme/<directory name>`.

## In CI

`gh fleet test` exits non-zero when any case fails, and `--json` prints one result per case:

```yaml
- run: gh fleet test -f campaigns/python-3.12/fleet.yaml
```

The [examples](../examples.md) in this repository use it: the ones with a `tests/` directory run their cases in the project's own test suite.
