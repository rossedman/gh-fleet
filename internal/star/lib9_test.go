package star

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLibBatchD(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]libCase{
		"codeql flow, block and input languages": {
			repo: map[string]string{
				".github/workflows/a.yml": "matrix:\n  language: [ 'javascript' ]\nsteps:\n  - uses: github/codeql-action/init@v3\n",
				".github/workflows/b.yml": "matrix:\n  language:\n    - go # main\n    - python\nsteps:\n  - uses: github/codeql-action/init@v3\n",
				".github/workflows/c.yml": "steps:\n  - uses: \"github/codeql-action/init@v3\"\n    with:\n      languages: ruby  # app\n",
				".github/workflows/d.yml": "steps:\n  - uses: actions/checkout@v4\n",
			},
			script: `load("@fleet//codeql.star", "add_language", "add_workflow", "languages")
def change(repo):
    repo.set_var("before", languages(repo))
    repo.set_var("ts", add_language(repo, "typescript"))
    repo.set_var("actions", add_language(repo, "actions"))
    repo.set_var("wrote", add_workflow(repo, ["go"]))
`,
			want: map[string]string{
				".github/workflows/a.yml": "matrix:\n  language: [ 'javascript', 'actions' ]\nsteps:\n  - uses: github/codeql-action/init@v3\n",
				".github/workflows/b.yml": "matrix:\n  language:\n    - go # main\n    - python\n    - typescript\n    - actions\nsteps:\n  - uses: github/codeql-action/init@v3\n",
				".github/workflows/c.yml": "steps:\n  - uses: \"github/codeql-action/init@v3\"\n    with:\n      languages: ruby, typescript, actions  # app\n",
			},
			vars: map[string]string{
				"before":  "[go javascript-typescript python ruby]",
				"ts":      "[.github/workflows/b.yml .github/workflows/c.yml]",
				"actions": "[.github/workflows/a.yml .github/workflows/b.yml .github/workflows/c.yml]",
				"wrote":   "false",
			},
		},
		"codeql include without build-mode": {
			repo: map[string]string{".github/workflows/codeql.yml": "matrix:\n  include:\n  - language: java\n  - language: cpp\nsteps:\n- uses: github/codeql-action/init@v3\n"},
			script: `load("@fleet//codeql.star", "add_language")
def change(repo):
    add_language(repo, "kotlin")
    add_language(repo, "c")
    add_language(repo, "swift")
`,
			want: map[string]string{".github/workflows/codeql.yml": "matrix:\n  include:\n  - language: java\n  - language: cpp\n  - language: swift\nsteps:\n- uses: github/codeql-action/init@v3\n"},
		},
		"scorecard published workflow and badges": {
			repo: map[string]string{"README.md": "# api\n\nText.\n", "docs/README.md": "no heading\n"},
			script: `load("@fleet//scorecard.star", "add_badge", "add_workflow", "workflow")
def change(repo):
    add_workflow(repo)
    repo.set_var("again", add_workflow(repo))
    repo.set_var("path", workflow(repo))
    add_badge(repo)
    repo.set_var("badge", add_badge(repo))
    add_badge(repo, "docs/README.md")
    repo.set_var("missing", add_badge(repo, "nope.md"))
`,
			want: map[string]string{
				"README.md":      "# api\n\n[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/acme/api/badge)](https://scorecard.dev/viewer/?uri=github.com/acme/api)\n\nText.\n",
				"docs/README.md": "[![OpenSSF Scorecard](https://api.scorecard.dev/projects/github.com/acme/api/badge)](https://scorecard.dev/viewer/?uri=github.com/acme/api)\n\nno heading\n",
			},
			vars: map[string]string{"again": "false", "path": ".github/workflows/scorecard.yml", "badge": "false", "missing": "false"},
		},
		"scanner ignore files": {
			repo: map[string]string{
				".gitleaksignore": "abc123:config/test.env:generic-api-key:3",
				".trivyignore":    "# old\nCVE-1 exp:2020-01-01\n\n# keep\nCVE-2\nCVE-3 exp:2020-01-01\n",
			},
			script: `load("@fleet//scanignore.star", "add", "entries", "prune_expired", "remove")
def change(repo):
    repo.set_var("g", add(repo, "gitleaks", ["def456:x:y:1", " def456:x:y:1 "]))
    repo.set_var("s", add(repo, "semgrep", ["vendor/"]))
    repo.set_var("pruned", prune_expired(repo, "2026-01-01"))
    repo.set_var("none", remove(repo, "semgrep", ["gen/"]))
    repo.set_var("absent", remove(repo, "gitleaks", ["zzz"]) + prune_expired(repo, "2026-01-01"))
    repo.set_var("left", entries(repo, "trivy"))
`,
			want: map[string]string{
				".gitleaksignore": "abc123:config/test.env:generic-api-key:3\n\ndef456:x:y:1\n",
				".semgrepignore":  "vendor/\n",
				".trivyignore":    "# keep\nCVE-2\n",
			},
			vars: map[string]string{"g": "[def456:x:y:1]", "s": "[vendor/]", "pruned": "[CVE-1 CVE-3]", "none": "[]", "absent": "[]", "left": "[CVE-2]"},
		},
		"emptied ignore file": {
			repo:   map[string]string{".semgrepignore": "# all generated\ngen/\n"},
			script: "load('@fleet//scanignore.star', 'remove')\ndef change(repo):\n    remove(repo, 'semgrep', ['gen/'])\n",
			want:   map[string]string{".semgrepignore": ""},
		},
	} {
		runLib(t, name, tc)
	}

	for script, want := range map[string]string{
		"load('@fleet//scanignore.star', 'entries')\ndef change(repo):\n  entries(repo, 'snyk')\n":                           `unknown tool "snyk"; have gitleaks, semgrep, trivy`,
		"load('@fleet//scanignore.star', 'add')\ndef change(repo):\n  add(repo, 'semgrep', ['x'], expires = '2026-01-01')\n": "only trivy supports expires",
		"load('@fleet//scanignore.star', 'add')\ndef change(repo):\n  add(repo, 'trivy', ['x'], expires = 'soon')\n":         "is not a YYYY-MM-DD date",
		"load('@fleet//scanignore.star', 'prune_expired')\ndef change(repo):\n  prune_expired(repo, '1/2/2026')\n":           "is not a YYYY-MM-DD date",
		"load('@fleet//codeql.star', 'add_workflow')\ndef change(repo):\n  add_workflow(repo, [])\n":                         "no languages given",
	} {
		if _, _, _, err := run(t, script, nil, RepoInfo{}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%q: got %v, want %q", script, err, want)
		}
	}
}

func TestCodeQLWorkflowBranch(t *testing.T) {
	t.Parallel()
	dir, _, _, err := run(t, "load('@fleet//codeql.star', 'add_workflow')\nload('@fleet//scorecard.star', sc = 'add_workflow')\ndef change(repo):\n  add_workflow(repo, ['go'], versions = {'actions/checkout': 'v6'})\n  sc(repo, publish = True)\n", nil, RepoInfo{Owner: "acme", Name: "api", DefaultBranch: "trunk"})
	if err != nil {
		t.Fatal(err)
	}
	codeql := readRepo(t, dir, ".github/workflows/codeql.yml")
	for _, s := range []string{"branches: [trunk]", "actions/checkout@v6", "github/codeql-action/init@v4", "- language: go\n            build-mode: autobuild"} {
		if !strings.Contains(codeql, s) {
			t.Errorf("codeql.yml missing %q:\n%s", s, codeql)
		}
	}
	sc := readRepo(t, dir, ".github/workflows/scorecard.yml")
	for _, s := range []string{"branches: [trunk]", "id-token: write", "publish_results: true"} {
		if !strings.Contains(sc, s) {
			t.Errorf("scorecard.yml missing %q:\n%s", s, sc)
		}
	}
}

func readRepo(t *testing.T, dir, path string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, path))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
