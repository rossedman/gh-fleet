//go:build integration

package app

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/state"
	"github.com/rossedman/gh-fleet/internal/target"
	"github.com/rossedman/gh-fleet/internal/workspace"
)

// TestIntegrationCampaign runs a full campaign against a real throwaway
// org: apply opens one PR, a second apply updates it without duplicating,
// status sees it after the state file is deleted, and close cleans up.
// It needs GH_FLEET_TEST_ORG, GH_FLEET_TEST_REPO (a repo in that org that
// may be changed) and a gh login with push access to it.
func TestIntegrationCampaign(t *testing.T) {
	org, repo := os.Getenv("GH_FLEET_TEST_ORG"), os.Getenv("GH_FLEET_TEST_REPO")
	if org == "" || repo == "" {
		t.Skip("set GH_FLEET_TEST_ORG and GH_FLEET_TEST_REPO to run against a real org")
	}
	ctx := context.Background()
	name := fmt.Sprintf("it-%d", time.Now().Unix())
	dir := t.TempDir()
	file := filepath.Join(dir, "fleet.yaml")
	writeIT := func(line string) {
		t.Helper()
		c := fmt.Sprintf(`apiVersion: fleet.gh/v1
name: %s
targets:
  include:
    repos: [%s]
change:
  run: echo %q > .gh-fleet-integration
pr:
  title: "test: gh fleet integration {{ .Repo.Name }}"
  body: Opened by the gh fleet integration suite; it closes itself.
  draft: true
`, name, repo, line)
		if err := os.WriteFile(file, []byte(c), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	writeIT("first")

	var out, errOut bytes.Buffer
	a := &App{
		Out: &out, Err: &errOut, In: strings.NewReader(""),
		NewClient: func(rate float64) (ghapi.Client, error) { return ghapi.New(ghapi.Options{WriteRate: rate}) },
		Store:     &state.Store{Dir: filepath.Join(dir, "state")},
		Cache:     &target.Cache{Dir: filepath.Join(dir, "cache")},
		Workspaces: &workspace.Manager{
			Root: filepath.Join(dir, "work"),
		},
	}
	c := Common{File: file, Yes: true}
	t.Cleanup(func() {
		_ = a.Close(context.Background(), CloseOptions{Common: c, Comment: "Integration run finished."})
	})

	if err := a.Apply(ctx, ApplyOptions{Common: c}); err != nil {
		t.Fatalf("apply: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(errOut.String(), "created") {
		t.Fatalf("first apply did not create a PR:\n%s", errOut.String())
	}

	errOut.Reset()
	writeIT("second")
	if err := a.Apply(ctx, ApplyOptions{Common: c}); err != nil {
		t.Fatalf("second apply: %v\n%s", err, errOut.String())
	}
	if strings.Contains(errOut.String(), "created") {
		t.Fatalf("second apply created another PR:\n%s", errOut.String())
	}

	if err := os.RemoveAll(a.Store.Dir); err != nil {
		t.Fatal(err)
	}
	out.Reset()
	if err := a.Status(ctx, Common{File: file, JSON: true}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), `"status": "pr_open"`) {
		t.Fatalf("status after state loss:\n%s", out.String())
	}

	errOut.Reset()
	if err := a.Close(ctx, CloseOptions{Common: c, Comment: "Integration run finished."}); err != nil {
		t.Fatalf("close: %v\n%s", err, errOut.String())
	}
	if !strings.Contains(errOut.String(), "1 closed") {
		t.Fatalf("close:\n%s", errOut.String())
	}
}
