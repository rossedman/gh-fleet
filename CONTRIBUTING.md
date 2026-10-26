# Contributing to gh-fleet

Thanks for helping. This page covers setting up, the checks every change must
pass, and how changes get merged.

## Set up

You need Go (the version in `go.mod`), git, and [uv](https://docs.astral.sh/uv/)
for the docs site. golangci-lint, govulncheck and goreleaser are run with
`go run` at pinned versions, so there is nothing else to install.

```sh
git clone https://github.com/rossedman/gh-fleet
cd gh-fleet
make build              # builds ./gh-fleet
gh extension install .  # use your build as `gh fleet`
```

## Run every check

```sh
make check
```

That runs, in order: build, `golangci-lint`, `go test -race` with the coverage
gate, `tools/check-cmds.sh`, doc comment checks, generated docs drift, a
benchmark smoke run, a fuzz smoke run, govulncheck, the strict docs build and
`goreleaser check`. CI runs the same targets. Smaller targets:

| Target | What it does |
| --- | --- |
| `make test` | `go test -race ./...` |
| `make cover` | tests with coverage, failing below 90% of hand-written code |
| `make lint` | golangci-lint, check-cmds, doccheck |
| `make bench` | every benchmark, results saved to `bench/` for `benchstat` |
| `make fuzz` | every fuzz target for `FUZZTIME` (default 30s) |
| `make docs` | regenerate `docs/reference/` from the CLI and the campaign schema |
| `make schema` | refresh GitHub's GraphQL schema, which every query the client sends is validated against |
| `make docs-serve` | preview the docs site at http://127.0.0.1:8000 |
| `make integration` | the opt-in suite against a real throwaway org (see below) |

Two more gates run in CI and, through `.claude/settings.json`, in Claude Code:

- [guardrails](https://github.com/rossedman/guardrails) (`.guardrails.yaml`,
  strict preset) blocks a Claude turn, and a pull request, until touched
  packages build, pass `go test -race`, meet 90% coverage (80% per package)
  and are lint and govulncheck clean. Run it yourself with
  `guardrails gate --all`.
- [sarifix](https://github.com/rossedman/sarifix) (`sarifix.yaml`) runs
  golangci-lint and gitleaks, reports what a change introduced with
  `sarifix check`, and on `main` uploads findings to code scanning and keeps
  one issue per batch of lint findings.

Both run golangci-lint built with this module's Go version, as `make lint`
does: the released binaries are built with an older Go and refuse to load
the module, so put a matching build on `PATH`
(`GOTOOLCHAIN=$(go env GOVERSION) go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.15.0`).

## Quality bars

- **Tests:** at least 90% coverage of hand-written code, enforced in CI. All
  GitHub access goes through the `internal/ghapi.Client` interface: test with
  the in-memory fake in `internal/ghapi/fake`, and test git against local bare
  repos from `internal/testutil`. No test talks to the real GitHub API; every
  GraphQL query the client builds is validated against GitHub's published
  schema in `internal/ghapi/testdata` instead. Tests run on Linux, macOS and
  Windows.
- **Doc comments** on every package, function and method, starting with the
  identifier's name (`go run ./tools/doccheck ./...`).
- **One cobra command per file** in `cmd/`, named after the command
  (`tools/check-cmds.sh`). Commands parse flags and call `internal/app`; no
  logic lives in `cmd/`.
- **Benchmarks** for every critical path and **fuzz targets** for every one
  that handles outside input (see below).
- **Terminal output** goes through `internal/ui` and `internal/render`: no
  colors defined anywhere else, nothing styled when output is piped or
  `--json` is set, and `NO_COLOR` is honored. Follow gh's conventions:
  results on stdout and everything else on stderr, tab-separated tables with
  no header when piped, and gh's exit codes (`docs/guides/scripting.md`).

## Critical paths

These functions parse input, sit in hot loops, or decide what gets pushed.
Changes to them need a benchmark and, where they take outside input, a fuzz
target.

| Function | Why | Benchmark | Fuzz |
| --- | --- | --- | --- |
| `campaign.Parse` | parses user YAML, schema validation, line-numbered errors | | `FuzzParse` |
| `target.Evaluate` | the include/exclude rules, run for every repo in an org | `BenchmarkEvaluate` | `FuzzEvaluate` |
| `target.Resolver.Resolve` | target resolution over large orgs | `BenchmarkResolve5000` | |
| `engine.Run` | the worker pool every command runs on | `BenchmarkEngineConcurrency{1,8,32}` | |
| `ghapi.Transport.RoundTrip` | rate limiting, pauses and retries for every API call | | |
| `engine.Pipeline.Process` | decides when to commit, push, open, update or close a PR | | |
| `star.localPath` | the sandbox boundary: every path a Starlark script passes to `repo` | | `FuzzLocalPath` |
| `star.Script.Run` | runs untrusted-by-default campaign code against a clone | | |
| `json.encode_like` | rewrites JSON files keeping their layout | | `FuzzEncodeLike` |
| `yamlpath.set` | line-based YAML edits under most library helpers | | `FuzzYAMLSet` |
| `toml.set` | line-based TOML edits | | `FuzzTOMLSet` |

## Adding to the Starlark library

The built-in library lives in `internal/star/lib/*.star`. Its reference
pages (`docs/reference/starlark-library.md` and `docs/reference/starlark/`)
and their entries in `mkdocs.yml` are generated from the docstrings by
`make docs`, so a docstring is the only documentation you write:

```python
def ensure_lines(repo, path, lines, header = None):
    """Appends each of `lines` that a file does not already contain.

    Optional prose: how it matches, what it leaves alone.

    Args:
        path: the file to edit, relative to the repo root.
        lines: a list of lines, or one string with a line per line.
        header: a line written above the new lines when any are added.
    Returns:
        The list of lines that were added.
    Example:
        before .gitignore:
            node_modules
        call:
            ensure_lines(repo, ".gitignore", [".env"])
        after .gitignore:
            node_modules
            .env
        returns:
            [".env"]
    """
```

- Every public function needs a summary line, an `Args:` entry for each
  parameter except `repo` (in order), `Returns:`, and an `Example:`.
  `TestLibDocumented` enforces this.
- An example lists files with `before <path>:` (and `campaign <path>:` for
  files next to the campaign), the `call:` (statements, then an
  expression), files with `after <path>:` or `deleted <path>`, and the
  `returns:` value as Starlark prints it. `TestLibExamples` runs every
  example, checks the files and the value, and runs it again to check that
  nothing is written the second time. Files with a `before` and no `after`
  must be left unchanged.
- Use `r"""` for a docstring that contains a backslash.
- Add table tests for the edge cases to `internal/star/lib*_test.go`, and an
  example campaign under `examples/` when the helper serves a common change;
  the reference links each function to the examples that use it.

## Integration tests

`make integration` runs a full campaign (apply twice, status, close) against a
real organization you own. It is skipped unless `GH_FLEET_TEST_ORG` names an
org whose repos may be changed, and `GH_FLEET_TEST_REPO` names one of them.
Use a throwaway org.

## Commits and pull requests

Commit messages and PR titles follow [Conventional Commits](https://www.conventionalcommits.org/):

```
feat(apply): add --reopen to reopen closed PRs
fix(target): treat topic filters as case-insensitive
docs: explain the commit-message override
```

The release changelog is grouped by these types, so pick the one that fits.

PRs are squash-merged once CI is green and a code owner has approved. Keep a
PR to one change; add or update tests and docs with it.

## Questions

Open a [discussion or issue](https://github.com/rossedman/gh-fleet/issues). For
security problems, follow the [security policy](https://github.com/rossedman/gh-fleet/blob/main/SECURITY.md) instead.
