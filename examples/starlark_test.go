package examples_test

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/star"
)

// runStar loads a Starlark campaign and runs its change in dir twice,
// failing if the second run changes anything, and returns the variables
// the first run set.
func runStar(t *testing.T, name, dir string) map[string]any {
	t.Helper()
	c, err := campaign.Load(filepath.Join(name, "fleet.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	s, err := star.Load(context.Background(), c.StarlarkFile(), c.Dir(), star.Options{Env: c.Change.Env})
	if err != nil {
		t.Fatal(err)
	}
	res, err := s.Run(context.Background(), dir, star.RepoInfo{Owner: "acme", Name: "web"}, nil, 0)
	if err != nil {
		t.Fatal(err)
	}
	before := snapshot(t, dir)
	if _, err := s.Run(context.Background(), dir, star.RepoInfo{Owner: "acme", Name: "web"}, nil, 0); err != nil {
		t.Fatal(err)
	}
	for f, content := range snapshot(t, dir) {
		if before[f] != content {
			t.Errorf("%s: second run changed %s", name, f)
		}
	}
	return res.Vars
}

func TestNodeEngines(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"package.json":                `{"name": "web", "version": "1.0.0", "engines": {"node": ">=18", "npm": ">=9"}}`,
		"packages/ui/package.json":    `{"name": "ui", "private": true}`,
		"packages/edge/package.json":  `{"name": "edge", "engines": {"node": ">=24"}}`,
		"node_modules/x/package.json": `{"name": "x"}`,
	})
	vars := runStar(t, "node-engines", dir)
	if got := read(t, filepath.Join(dir, "package.json")); got != "{\n  \"name\": \"web\",\n  \"version\": \"1.0.0\",\n  \"engines\": {\n    \"node\": \">=22\",\n    \"npm\": \">=9\"\n  }\n}\n" {
		t.Errorf("package.json =\n%s", got)
	}
	if got := read(t, filepath.Join(dir, "packages/ui/package.json")); !strings.Contains(got, `"node": ">=22"`) {
		t.Errorf("ui package.json =\n%s", got)
	}
	if got := read(t, filepath.Join(dir, "packages/edge/package.json")); strings.Contains(got, "22") {
		t.Errorf("a newer requirement was lowered:\n%s", got)
	}
	if got := read(t, filepath.Join(dir, "node_modules/x/package.json")); got != `{"name": "x"}` {
		t.Errorf("node_modules touched: %s", got)
	}
	if vars["node"] != int64(22) || vars["previous"] != ">=18, unset" {
		t.Errorf("vars = %v", vars)
	}
}
