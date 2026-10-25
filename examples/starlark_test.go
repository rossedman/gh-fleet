package examples_test

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestNodeEngines(t *testing.T) {
	t.Parallel()
	dir := scratch(t, map[string]string{
		"package.json":                `{"name": "web", "version": "1.0.0", "engines": {"node": ">=18", "npm": ">=9"}}`,
		"packages/ui/package.json":    `{"name": "ui", "private": true}`,
		"packages/edge/package.json":  `{"name": "edge", "engines": {"node": ">=24"}}`,
		"node_modules/x/package.json": `{"name": "x"}`,
	})
	vars := runTwice(t, "node-engines", dir)
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
