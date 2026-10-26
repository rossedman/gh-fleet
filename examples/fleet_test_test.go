package examples_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/rossedman/gh-fleet/internal/app"
	"github.com/rossedman/gh-fleet/internal/gitops"
)

// TestFleetTestCases runs gh fleet test on every example that has test
// cases, the way a campaign author would.
func TestFleetTestCases(t *testing.T) {
	t.Parallel()
	dirs, _ := filepath.Glob("*/" + app.TestDir)
	if len(dirs) < 10 {
		t.Fatalf("want test cases in at least 10 examples, got %v", dirs)
	}
	for _, d := range dirs {
		name := filepath.Dir(d)
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			var out bytes.Buffer
			a := &app.App{Out: &out, Err: &out, Git: gitops.Git{}, LibCache: t.TempDir()}
			err := a.Test(context.Background(), app.TestOptions{Common: app.Common{File: filepath.Join(name, "fleet.yaml")}})
			if err != nil {
				t.Errorf("%v\n%s", err, out.String())
			}
		})
	}
	if _, err := os.Stat("trivy-ignore-prune/tests/expired/after/.trivyignore"); err != nil {
		t.Error(err)
	}
}
