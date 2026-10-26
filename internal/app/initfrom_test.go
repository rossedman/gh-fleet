package app

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitFrom(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	var out, errOut bytes.Buffer
	a := &App{Out: &out, Err: &errOut}
	file := filepath.Join(dir, "campaigns", "node", "fleet.yaml")
	if err := a.Init(InitOptions{From: "examples/node-pnpm-9/", Name: "Pnpm Nine", Org: "mycorp", File: file}); err != nil {
		t.Fatal(err)
	}
	b, _ := os.ReadFile(file)
	if !strings.Contains(string(b), "\nname: pnpm-nine\n") || !strings.Contains(string(b), "orgs: [mycorp]") {
		t.Errorf("fleet.yaml:\n%s", b)
	}
	for _, p := range []string{"change.star", "tests/pnpm-8/before/package.json", "tests/pnpm-8/after/.npmrc"} {
		if _, err := os.Stat(filepath.Join(dir, "campaigns", "node", p)); err != nil {
			t.Errorf("not copied: %v", err)
		}
	}
	if !strings.Contains(out.String(), "Copied 7 files from example node-pnpm-9") {
		t.Errorf("output = %q", out.String())
	}

	// The copy runs against its own test cases.
	h := newHarness(t)
	if err := h.app.Test(t.Context(), TestOptions{Common: Common{File: file}}); err != nil {
		t.Errorf("copied tests fail: %v\n%s", err, h.out)
	}

	// Refused before writing anything when a file exists; --force overwrites.
	if err := os.WriteFile(filepath.Join(dir, "campaigns", "node", "change.star"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.Init(InitOptions{From: "node-pnpm-9", File: file}); err == nil || !strings.Contains(err.Error(), "change.star already exists") {
		t.Fatalf("err = %v", err)
	}
	if b, _ := os.ReadFile(filepath.Join(dir, "campaigns", "node", "change.star")); string(b) != "mine" {
		t.Error("existing file overwritten")
	}
	if err := a.Init(InitOptions{From: "node-pnpm-9", File: file, Force: true}); err != nil {
		t.Fatal(err)
	}
	if b, _ := os.ReadFile(file); !strings.Contains(string(b), "name: node-pnpm-9") {
		t.Error("--force without a name should keep the example's name")
	}

	for from, want := range map[string]string{"pyhton-version": "close names: python-version", "zzz": "gh fleet init --list shows them"} {
		if err := a.Init(InitOptions{From: from, File: filepath.Join(dir, "x", "fleet.yaml")}); err == nil || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: err = %v", from, err)
		}
	}

	out.Reset()
	errOut.Reset()
	if err := a.Init(InitOptions{List: true}); err != nil || !strings.Contains(out.String(), "node-pnpm-9") || !strings.Contains(out.String(), "EXAMPLE") || !strings.Contains(errOut.String(), "--from <example>") {
		t.Errorf("list: %v\n%s", err, out.String())
	}
	if got := campaignDescription([]byte("name: x\n")); got != "" {
		t.Errorf("no description = %q", got)
	}
}
