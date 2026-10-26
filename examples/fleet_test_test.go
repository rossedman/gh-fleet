package examples_test

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/examples"

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

// TestEmbedded checks that every example file ships in the binary, for
// gh fleet init --from.
func TestEmbedded(t *testing.T) {
	t.Parallel()
	if got := len(examples.Names()); got != 81 {
		t.Fatalf("embedded %d examples", got)
	}
	err := filepath.WalkDir(".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.Contains(p, string(filepath.Separator)) {
			return err
		}
		if _, err := fs.Stat(examples.FS, filepath.ToSlash(p)); err != nil {
			t.Errorf("%s is not embedded; add its pattern to embed.go", p)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
