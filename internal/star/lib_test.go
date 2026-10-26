package star

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"go.starlark.net/starlark"
)

// libCase runs a script that uses the built-in library against a repo,
// checks the files it leaves, and runs it again to check that a second run
// writes nothing.
type libCase struct {
	campaign map[string]string
	repo     map[string]string
	script   string
	want     map[string]string
	gone     []string
	vars     map[string]string
}

// runLib runs one libCase.
func runLib(t *testing.T, name string, tc libCase) {
	t.Helper()
	camp := map[string]string{"change.star": tc.script}
	for k, v := range tc.campaign {
		camp[k] = v
	}
	s, err := load(t, camp, Options{})
	if err != nil {
		t.Fatalf("%s: load: %v", name, err)
	}
	dir := files(t, tc.repo)
	for i := range 2 {
		var log bytes.Buffer
		res, err := s.Run(context.Background(), dir, RepoInfo{Owner: "acme", Name: "api"}, &log, time.Minute)
		if err != nil {
			t.Fatalf("%s: run %d: %v", name, i+1, err)
		}
		if i == 1 && strings.Contains(log.String(), "starlark: ") {
			t.Errorf("%s: second run wrote files:\n%s", name, log.String())
		}
		if i == 0 {
			for k, want := range tc.vars {
				if got := fmtAny(res.Vars[k]); got != want {
					t.Errorf("%s: var %s = %s, want %s", name, k, got, want)
				}
			}
		}
	}
	for path, want := range tc.want {
		got, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Errorf("%s: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("%s: %s =\n%s\nwant\n%s", name, path, got, want)
		}
	}
	for _, path := range tc.gone {
		if _, err := os.Stat(filepath.Join(dir, path)); err == nil {
			t.Errorf("%s: %s still exists", name, path)
		}
	}
}

func TestLibFiles(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"ensure_lines": {
			repo: map[string]string{".gitignore": "node_modules\n.env"},
			script: `load("@fleet//files.star", "ensure_lines")
def change(repo):
    repo.set_var("added", ensure_lines(repo, ".gitignore", ".env\n\n.env.local\n.envrc", header = "# Local"))
    repo.set_var("created", ensure_lines(repo, "new.txt", ["a"]))
`,
			want: map[string]string{".gitignore": "node_modules\n.env\n\n# Local\n.env.local\n.envrc\n", "new.txt": "a\n"},
			vars: map[string]string{"added": "[.env.local .envrc]", "created": "[a]"},
		},
		"remove_lines": {
			repo: map[string]string{".gitignore": "a\n b \nc\n", "only.txt": "x\n"},
			script: `load("@fleet//files.star", "remove_lines")
def change(repo):
    repo.set_var("removed", remove_lines(repo, ".gitignore", ["b", "zz"]))
    remove_lines(repo, "only.txt", "x")
    remove_lines(repo, "missing.txt", "x")
`,
			want: map[string]string{".gitignore": "a\nc\n", "only.txt": ""},
			vars: map[string]string{"removed": "[b]"},
		},
		"replace_all": {
			repo: map[string]string{"a.md": "see old.example.com", "b/c.go": "old.example.com/x", "bin": "old.example.com\x00", "vendor/v.go": "old.example.com"},
			script: `load("@fleet//files.star", "replace_all")
def change(repo):
    repo.set_var("changed", replace_all(repo, "old.example.com", "new.example.com", exclude = ["vendor/**"]))
`,
			want: map[string]string{"a.md": "see new.example.com", "b/c.go": "new.example.com/x", "bin": "old.example.com\x00", "vendor/v.go": "old.example.com"},
			vars: map[string]string{"changed": "[a.md b/c.go]"},
		},
		"write_if_missing": {
			repo: map[string]string{"CODEOWNERS": "* @x\n"},
			script: `load("@fleet//files.star", "write_if_missing")
def change(repo):
    repo.set_var("a", write_if_missing(repo, ".github/CODEOWNERS", "* @y\n", also = ["CODEOWNERS"]))
    repo.set_var("b", write_if_missing(repo, "LICENSE", "MIT\n"))
`,
			want: map[string]string{"CODEOWNERS": "* @x\n", "LICENSE": "MIT\n"},
			vars: map[string]string{"a": "false", "b": "true"},
		},
		"sync": {
			campaign: map[string]string{"files/.editorconfig": "root = true\n", "files/.github/pull_request_template.md": "## What\n", "files/skip.tmp": "x"},
			repo:     map[string]string{".editorconfig": "old\n", ".github/pull_request_template.md": "## What\n", "keep.txt": "k"},
			script: `load("@fleet//files.star", "sync")
def change(repo):
    repo.set_var("changed", sync(repo, "./files/", exclude = ["**/*.tmp"]))
    sync(repo, "files", dest = "templates")
`,
			want: map[string]string{".editorconfig": "root = true\n", "keep.txt": "k", "templates/skip.tmp": "x"},
			gone: []string{"skip.tmp"},
			vars: map[string]string{"changed": "[.editorconfig]"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibGHA(t *testing.T) {
	t.Parallel()
	wf := `name: ci
on: push
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3  # keep
      - uses: "actions/checkout@v3.1"
      - uses: actions/checkout-extra@v3
      - uses: actions/cache/restore@v3
      - uses: actions/checkout@0123456789abcdef0123456789abcdef01234567
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - uses: actions/setup-go@v5
        with:
          go-version: stable
`
	for name, tc := range map[string]libCase{
		"bump_action": {
			repo: map[string]string{".github/workflows/ci.yml": wf, "act/action.yml": "runs:\n  steps:\n    - uses: actions/checkout@v3\n", ".github/workflows/other.yaml": "on: push\n"},
			script: `load("@fleet//gha.star", "bump_action", "set_input")
def change(repo):
    repo.set_var("changed", bump_action(repo, "actions/checkout", "v4", from_ref = "v3"))
    bump_action(repo, "actions/cache", "v4")
    set_input(repo, "go-version", "1.27", match = r"^\d")
`,
			want: map[string]string{
				".github/workflows/ci.yml": strings.NewReplacer(
					"actions/checkout@v3  #", "actions/checkout@v4  #",
					"actions/cache/restore@v3", "actions/cache/restore@v4",
					"'1.22'", "'1.27'",
				).Replace(wf),
				"act/action.yml": "runs:\n  steps:\n    - uses: actions/checkout@v4\n",
			},
			vars: map[string]string{"changed": "[.github/workflows/ci.yml act/action.yml]"},
		},
		"ensure_permissions": {
			repo: map[string]string{
				".github/workflows/a.yml": "on: push\njobs:\n  x: {}\n",
				".github/workflows/b.yml": "on: push\npermissions: write-all\njobs:\n  x: {}\n",
				".github/workflows/c.yml": "jobs:\n  x: {}\n",
				".github/workflows/d.yml": "on: push\n",
			},
			script: `load("@fleet//gha.star", "ensure_permissions", "workflows")
def change(repo):
    repo.set_var("changed", ensure_permissions(repo))
    repo.set_var("count", len(workflows(repo)))
`,
			want: map[string]string{
				".github/workflows/a.yml": "on: push\npermissions:\n  contents: read\n\njobs:\n  x: {}\n",
				".github/workflows/b.yml": "on: push\npermissions: write-all\njobs:\n  x: {}\n",
				".github/workflows/c.yml": "permissions:\n  contents: read\n\njobs:\n  x: {}\n",
				".github/workflows/d.yml": "on: push\n",
			},
			vars: map[string]string{"changed": "[.github/workflows/a.yml .github/workflows/c.yml]", "count": "4"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibEcosystems(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"docker": {
			repo: map[string]string{
				"Dockerfile":                "FROM --platform=$BUILDPLATFORM node:18-alpine AS build\nfrom node:18\nFROM node:18.2\nFROM node@sha256:abc\n",
				"web/app.Dockerfile":        "FROM node:18 AS x\n",
				"node_modules/x/Dockerfile": "FROM node:18\n",
			},
			script: `load("@fleet//docker.star", "bump_base_image")
def change(repo):
    repo.set_var("changed", bump_base_image(repo, "node", "18", "22"))
`,
			want: map[string]string{
				"Dockerfile":                "FROM --platform=$BUILDPLATFORM node:22-alpine AS build\nfrom node:22\nFROM node:18.2\nFROM node@sha256:abc\n",
				"web/app.Dockerfile":        "FROM node:22 AS x\n",
				"node_modules/x/Dockerfile": "FROM node:18\n",
			},
			vars: map[string]string{"changed": "[Dockerfile web/app.Dockerfile]"},
		},
		"docker replace any tag": {
			repo: map[string]string{
				"Dockerfile": "FROM --platform=$BUILDPLATFORM python:3.12-slim AS build\nfrom python\nFROM python@sha256:abc AS pinned\nFROM python:3.12@sha256:def\n" +
					"FROM python-dev:1\nFROM python/tools:2\nFROM mypython:3\nFROM ghcr.io/python:1\nRUN echo FROM python:3\n",
				"svc/Dockerfile.prod": "FROM localhost:5000/base:1\n",
			},
			script: `load("@fleet//docker.star", "replace_base_image")
def change(repo):
    repo.set_var("changed", replace_base_image(repo, "python", "registry.acme.io/py/3.14-slim:main-$x"))
    replace_base_image(repo, "localhost:5000/base", "base:2")
`,
			want: map[string]string{
				"Dockerfile": "FROM --platform=$BUILDPLATFORM registry.acme.io/py/3.14-slim:main-$x AS build\nfrom registry.acme.io/py/3.14-slim:main-$x\n" +
					"FROM registry.acme.io/py/3.14-slim:main-$x AS pinned\nFROM registry.acme.io/py/3.14-slim:main-$x\n" +
					"FROM python-dev:1\nFROM python/tools:2\nFROM mypython:3\nFROM ghcr.io/python:1\nRUN echo FROM python:3\n",
				"svc/Dockerfile.prod": "FROM base:2\n",
			},
			vars: map[string]string{"changed": "[Dockerfile]"},
		},
		"docker replace one tag": {
			repo: map[string]string{
				"Dockerfile": "FROM node:18\nFROM node:18-alpine\nFROM node:20\nFROM ghcr.io/acme/node@sha256:1 AS x\n",
			},
			script: `load("@fleet//docker.star", "replace_base_image")
def change(repo):
    replace_base_image(repo, "node:18", "registry.acme.io/node:22")
    replace_base_image(repo, "ghcr.io/acme/node@sha256:1", "registry.acme.io/node:22")
`,
			want: map[string]string{
				"Dockerfile": "FROM registry.acme.io/node:22\nFROM node:18-alpine\nFROM node:20\nFROM registry.acme.io/node:22 AS x\n",
			},
		},
		"gomod": {
			repo: map[string]string{
				"go.mod":            "module a\n\ngo 1.21\n\ntoolchain go1.22.1\n\nrequire x v1\n",
				"new/go.mod":        "module b\n\ngo 1.28\n",
				"tc/go.mod":         "module c\n\ngo 1.20\ntoolchain go1.30.0\n",
				"end/go.mod":        "module f\n\ngo 1.22.1\n\ntoolchain go1.22.5\n",
				"none/go.mod":       "module d\n",
				"tight/go.mod":      "module e\ngo 1.21\ntoolchain go1.22\nrequire x v1\n",
				"vendor/x/go.mod":   "module v\n\ngo 1.10\n",
				"x/testdata/go.mod": "module t\n\ngo 1.10\n",
			},
			script: `load("@fleet//gomod.star", "go_version", "set_go_version")
def change(repo):
    repo.set_var("changed", set_go_version(repo, "1.27.3"))
    repo.set_var("root", go_version(repo))
    repo.set_var("missing", go_version(repo, "nope/go.mod") == None)
`,
			want: map[string]string{
				"go.mod":            "module a\n\ngo 1.27.3\n\nrequire x v1\n",
				"new/go.mod":        "module b\n\ngo 1.28\n",
				"tc/go.mod":         "module c\n\ngo 1.27.3\ntoolchain go1.30.0\n",
				"end/go.mod":        "module f\n\ngo 1.27.3\n",
				"none/go.mod":       "module d\n\ngo 1.27.3\n",
				"tight/go.mod":      "module e\ngo 1.27.3\nrequire x v1\n",
				"vendor/x/go.mod":   "module v\n\ngo 1.10\n",
				"x/testdata/go.mod": "module t\n\ngo 1.10\n",
			},
			vars: map[string]string{"changed": "[end/go.mod go.mod none/go.mod tc/go.mod tight/go.mod]", "root": "1.27.3", "missing": "true"},
		},
		"npm": {
			repo: map[string]string{
				"package.json":                    "{\n    \"name\": \"a\",\n    \"dependencies\": {\n        \"zod\": \"^3.0.0\",\n        \"axios\": \"^1.6.0\"\n    }\n}\n",
				"web/package.json":                "{\n  \"devDependencies\": {\"axios\": \"^1.7.9\"}\n}\n",
				"lib/package.json":                "{\"name\": \"lib\"}\n",
				"node_modules/axios/package.json": "{\"dependencies\": {\"axios\": \"0\"}}\n",
			},
			script: `load("@fleet//npm.star", "read_package", "set_dependency", "write_package")
def change(repo):
    repo.set_var("changed", set_dependency(repo, "axios", "^1.7.9"))
    pkg = read_package(repo, "lib/package.json")
    pkg["private"] = True
    write_package(repo, "new/package.json", pkg)
`,
			want: map[string]string{
				"package.json":     "{\n    \"name\": \"a\",\n    \"dependencies\": {\n        \"zod\": \"^3.0.0\",\n        \"axios\": \"^1.7.9\"\n    }\n}\n",
				"web/package.json": "{\n  \"devDependencies\": {\"axios\": \"^1.7.9\"}\n}\n",
				"lib/package.json": "{\"name\": \"lib\"}\n",
				"new/package.json": "{\n  \"name\": \"lib\",\n  \"private\": true\n}\n",
			},
			vars: map[string]string{"changed": "[package.json]"},
		},
		"codeowners new": {
			repo: map[string]string{},
			script: `load("@fleet//codeowners.star", "owners", "set_owners")
def change(repo):
    repo.set_var("before", owners(repo))
    set_owners(repo, "*", "acme/payments", header = "# Owners")
    repo.set_var("after", owners(repo))
`,
			want: map[string]string{".github/CODEOWNERS": "# Owners\n* @acme/payments\n"},
			vars: map[string]string{"before": "[]", "after": "[@acme/payments]"},
		},
		"codeowners edit": {
			repo: map[string]string{"docs/CODEOWNERS": "* @old # was\n/docs @writers", "CODEOWNERS": ""},
			script: `load("@fleet//codeowners.star", "find", "set_owners")
def change(repo):
    repo.set_var("path", find(repo))
    set_owners(repo, "*", ["@a", "b@example.com"])
    set_owners(repo, "/api", ["c"])
`,
			want: map[string]string{"CODEOWNERS": "* @a b@example.com\n/api @c\n", "docs/CODEOWNERS": "* @old # was\n/docs @writers"},
			vars: map[string]string{"path": "CODEOWNERS"},
		},
		"codeowners replace": {
			repo: map[string]string{".github/CODEOWNERS": "* @old # was\n/docs @writers"},
			script: `load("@fleet//codeowners.star", "set_owners")
def change(repo):
    set_owners(repo, "*", "@new")
    set_owners(repo, "/web", "@web")
`,
			want: map[string]string{".github/CODEOWNERS": "* @new\n/docs @writers\n/web @web\n"},
		},
		"dependabot": {
			repo: map[string]string{"go.mod": "module a\n", ".github/workflows/ci.yml": "on: push\n"},
			script: `load("@fleet//dependabot.star", "write_if_missing")
def change(repo):
    repo.set_var("written", write_if_missing(repo, prefixes = {"gomod": "deps"}))
`,
			want: map[string]string{".github/dependabot.yml": `version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
    groups:
      minor-and-patch:
        update-types: [minor, patch]
    commit-message:
      prefix: "deps"
  - package-ecosystem: github-actions
    directory: /
    schedule:
      interval: weekly
    groups:
      minor-and-patch:
        update-types: [minor, patch]
    commit-message:
      prefix: "ci(deps)"
`},
			vars: map[string]string{"written": "[gomod github-actions]"},
		},
		"dependabot existing": {
			repo: map[string]string{"go.mod": "module a\n", ".github/dependabot.yaml": "version: 2\n"},
			script: `load("@fleet//dependabot.star", "config", "ecosystems", "write_if_missing")
def change(repo):
    repo.set_var("written", write_if_missing(repo))
    repo.set_var("found", ecosystems(repo))
    repo.set_var("plain", config(["npm"], interval = "daily", group = False))
`,
			want: map[string]string{".github/dependabot.yaml": "version: 2\n"},
			vars: map[string]string{"written": "[]", "found": "[gomod]", "plain": "version: 2\nupdates:\n  - package-ecosystem: npm\n    directory: /\n    schedule:\n      interval: daily\n    commit-message:\n      prefix: \"build(deps)\"\n"},
		},
	} {
		runLib(t, name, tc)
	}
}

func TestLibErrors(t *testing.T) {
	t.Parallel()
	for name, tt := range map[string]struct {
		files map[string]string
		want  string
	}{
		"unknown module": {map[string]string{"change.star": "load('@fleet//nope.star', 'x')\n"}, "no built-in module nope.star; have "},
		"nested":         {map[string]string{"change.star": "load('@fleet//a/b.star', 'x')\n"}, "no built-in module a/b.star"},
		"bad symbol":     {map[string]string{"change.star": "load('@fleet//files.star', 'nope')\ndef change(repo):\n  pass\n"}, "nope"},
	} {
		if _, err := load(t, tt.files, Options{}); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
	if _, err := load(t, map[string]string{"change.star": "load('@fleet//nope.star', 'x')\n"}, Options{}); err == nil || !strings.Contains(err.Error(), ", files.star, ") {
		t.Errorf("unknown module error does not list the modules: %v", err)
	}
	s := &Script{modules: map[string]*module{}, byName: map[string]string{}}
	if got, root, err := s.resolveLoad(context.Background(), "./files.star", LibPrefix+"gha.star", LibPrefix); err != nil || got != LibPrefix+"files.star" || root != LibPrefix {
		t.Errorf("relative load in library = %q, %q, %v", got, root, err)
	}
	if _, _, err := s.resolveLoad(context.Background(), "../x.star", LibPrefix+"gha.star", LibPrefix); err == nil {
		t.Error("library loaded a file outside itself")
	}

	for name, tt := range map[string]struct{ script, want string }{
		"replace empty": {"load('@fleet//files.star', 'replace_all')\ndef change(repo):\n  replace_all(repo, '', 'x')\n", "old must not be empty"},
		"sync root":     {"load('@fleet//files.star', 'sync')\ndef change(repo):\n  sync(repo, '.')\n", "src must be a directory"},
		"sync empty":    {"load('@fleet//files.star', 'sync')\ndef change(repo):\n  sync(repo, 'nothing')\n", "no files under nothing"},
		"campaign path": {"def change(repo):\n  campaign.read('../x')\n", `campaign.read: "../x" is not a path inside the campaign directory`},
	} {
		if _, _, _, err := run(t, tt.script, nil, RepoInfo{}); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%s: err = %v, want %q", name, err, tt.want)
		}
	}
}

func TestCampaignValue(t *testing.T) {
	t.Parallel()
	c := &campaignDir{dir: t.TempDir()}
	if c.String() != "<campaign>" || c.Type() != "campaign" || !bool(c.Truth()) || strings.Join(c.AttrNames(), ",") != "exists,glob,read" {
		t.Fatal("campaign value basics")
	}
	if _, err := c.Hash(); err == nil {
		t.Fatal("campaign hashed")
	}
	c.Freeze()
	if v, _ := c.Attr("write"); v != nil {
		t.Fatal("campaign can write")
	}
	gone := &campaignDir{dir: filepath.Join(c.dir, "missing")}
	read, _ := gone.Attr("read")
	if _, err := starlark.Call(&starlark.Thread{}, read, starlark.Tuple{starlark.String("x")}, nil); err == nil {
		t.Fatal("read from a missing campaign directory")
	}
}
