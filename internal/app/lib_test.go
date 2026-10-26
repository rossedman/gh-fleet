package app

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/star"
)

// lib runs Lib and returns stdout, stderr and the error.
func lib(t *testing.T, o LibOptions) (string, string, error) {
	t.Helper()
	var out, errOut bytes.Buffer
	a := &App{Out: &out, Err: &errOut}
	err := a.Lib(o)
	return out.String(), errOut.String(), err
}

func TestLib(t *testing.T) {
	t.Parallel()
	out, errOut, err := lib(t, LibOptions{})
	if err != nil || !strings.Contains(out, "MODULE") || !strings.Contains(out, "docker  ") || !strings.Contains(errOut, "modules. gh fleet lib <module>") {
		t.Fatalf("list: %v\n%s%s", err, out, errOut)
	}
	if n := strings.Count(out, "\n"); n != len(star.LibModules())+1 {
		t.Errorf("list has %d lines for %d modules", n, len(star.LibModules()))
	}

	for _, args := range [][]string{{"docker"}, {"docker.star"}, {"@fleet//docker.star"}} {
		out, _, err = lib(t, LibOptions{Args: args})
		if err != nil || !strings.Contains(out, "@fleet//docker.star\n") || !strings.Contains(out, `load("@fleet//docker.star", "dockerfiles", "bump_base_image", "bump_image")`) ||
			!strings.Contains(out, "\nbump_base_image(repo, image, from_tag, to_tag)\n    Moves `FROM` lines") || !strings.Contains(out, "gh fleet lib docker.<function>") {
			t.Fatalf("%v: %v\n%s", args, err, out)
		}
	}

	for _, args := range [][]string{{"docker.bump_base_image"}, {"docker", "bump_base_image"}, {"docker.star.bump_base_image"}} {
		out, _, err = lib(t, LibOptions{Args: args})
		for _, want := range []string{"bump_base_image(repo, image, from_tag, to_tag)\n", `load("@fleet//docker.star", "bump_base_image")`, "Arguments\n  image: ", "Returns\n  The list of paths that changed.", "before Dockerfile:\n    FROM node:18-alpine", "call:\n    bump_base_image(repo, \"node\", \"18\", \"22\")", "returns:\n    [\"Dockerfile\"]"} {
			if err != nil || !strings.Contains(out, want) {
				t.Fatalf("%v: %v, missing %q:\n%s", args, err, want, out)
			}
		}
	}
	// Multi-line argument docs and deleted files.
	out, _, err = lib(t, LibOptions{Args: []string{"files.sync"}})
	if err != nil || !strings.Contains(out, "Example") {
		t.Fatalf("files.sync: %v\n%s", err, out)
	}

	out, _, err = lib(t, LibOptions{Args: []string{"refs"}, Source: true})
	src, _ := star.LibSource("refs.star")
	if err != nil || out != string(src) {
		t.Fatalf("source: %v", err)
	}

	out, _, err = lib(t, LibOptions{Search: "Docker FROM"})
	if err != nil || !strings.HasPrefix(strings.Split(out, "\n")[1], "docker.bump_base_image") {
		t.Fatalf("search: %v\n%s", err, out)
	}
}

func TestLibJSON(t *testing.T) {
	t.Parallel()
	out, _, err := lib(t, LibOptions{Common: Common{JSON: true}})
	var mods []LibModule
	if err != nil || json.Unmarshal([]byte(out), &mods) != nil || len(mods) != len(star.LibModules()) || mods[0].Summary == "" {
		t.Fatalf("modules: %v %s", err, out)
	}
	out, _, err = lib(t, LibOptions{Common: Common{JSON: true}, Args: []string{"gha"}})
	var d star.LibDoc
	if err != nil || json.Unmarshal([]byte(out), &d) != nil || d.Name != "gha.star" || len(d.Funcs) == 0 {
		t.Fatalf("module: %v %s", err, out)
	}
	out, _, err = lib(t, LibOptions{Common: Common{JSON: true}, Args: []string{"gha.pin"}})
	var f star.LibFunc
	if err != nil || json.Unmarshal([]byte(out), &f) != nil || f.Name != "pin" || f.Example == nil {
		t.Fatalf("function: %v %s", err, out)
	}
	out, _, err = lib(t, LibOptions{Common: Common{JSON: true}, Search: "zzzz"})
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty search: %v %q", err, out)
	}
}

func TestLibErrors(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		o    LibOptions
		want string
	}{
		{LibOptions{Args: []string{"dokcer"}}, "no module dokcer in the library; close names: docker"},
		{LibOptions{Args: []string{"xyz"}}, "gh fleet lib lists them"},
		{LibOptions{Args: []string{"docker.nope"}}, "no function nope in docker.star; it has dockerfiles, bump_base_image, bump_image"},
		{LibOptions{Args: []string{"docker.bump_image", "x"}}, "name the function once"},
		{LibOptions{Args: []string{"a", "b", "c"}}, "at most one function"},
		{LibOptions{Search: "x", Args: []string{"docker"}}, "--search takes no module"},
		{LibOptions{Search: "zzzz"}, `nothing in the library mentions "zzzz"`},
		{LibOptions{Source: true}, "--source needs a module"},
		{LibOptions{Source: true, Args: []string{"docker.bump_image"}}, "drop .bump_image"},
	} {
		if _, _, err := lib(t, tt.o); err == nil || !strings.Contains(err.Error(), tt.want) {
			t.Errorf("%+v: err = %v, want %q", tt.o, err, tt.want)
		}
	}
}

func TestEditDistance(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		a, b string
		want int
	}{{"", "", 0}, {"docker", "dokcer", 2}, {"helm", "helmm", 1}, {"", "abc", 3}} {
		if got := editDistance(tt.a, tt.b); got != tt.want {
			t.Errorf("editDistance(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
