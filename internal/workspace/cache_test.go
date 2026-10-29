package workspace

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/testutil"
	"github.com/rossedman/gh-fleet/internal/timing"
)

// cachedManager returns a manager cloning remote through a git cache, with
// git commands timed in rec.
func cachedManager(t *testing.T, remote *testutil.Remote, env []string) (*Manager, *timing.Recorder) {
	t.Helper()
	rec := timing.New(nil)
	g := testutil.Git(env)
	g.Timings = rec
	return &Manager{
		Root: t.TempDir(), Git: g, CacheDir: t.TempDir(),
		CloneURL: func(ghapi.Repo) string { return remote.URL() },
	}, rec
}

// count returns how many times rec timed name.
func count(rec *timing.Recorder, name string) int {
	for _, s := range rec.Stats() {
		if s.Name == name {
			return s.Count
		}
	}
	return 0
}

func TestCachedClone(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	m, rec := cachedManager(t, remote, env)
	repo := ghapi.Repo{Owner: "acme", Name: "api"}

	ws, err := m.Clone(ctx, "c", repo, "main")
	if err != nil {
		t.Fatal(err)
	}
	// The first clone fills the cache instead of cloning.
	if count(rec, "git clone") != 0 || count(rec, "git fetch") != 1 {
		t.Fatalf("calls = %+v", rec.Stats())
	}
	cache := m.cachePath("acme", "api", remote.URL())
	if b, err := os.ReadFile(filepath.Join(ws.Dir, "a.txt")); err != nil || string(b) != "a\n" {
		t.Fatalf("a.txt = %q, %v", b, err)
	}
	if st, err := os.Stat(filepath.Join(ws.Dir, ".git")); err != nil || !st.IsDir() {
		t.Fatalf(".git is not a directory: %v", err)
	}
	if alt, _ := os.ReadFile(filepath.Join(ws.Dir, ".git", "objects", "info", "alternates")); !strings.Contains(filepath.FromSlash(string(alt)), filepath.Join(cache, "objects")) {
		t.Fatalf("alternates = %q", alt)
	}
	for args, want := range map[string]string{
		"config --get remote.origin.url":             remote.URL(),
		"rev-parse --abbrev-ref HEAD":                "main",
		"rev-parse --is-shallow-repository":          "true",
		"rev-parse origin/main":                      remote.Branch("main"),
		"rev-parse --abbrev-ref --symbolic main@{u}": "origin/main",
	} {
		if got, err := m.Git.Run(ctx, ws.Dir, strings.Fields(args)...); err != nil || got != want {
			t.Errorf("git %s = %q, %v; want %q", args, got, err, want)
		}
	}
	// What the workspace writes stays in the workspace.
	if _, err := m.Git.Run(ctx, ws.Dir, "config", "user.name", "script"); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Dir, "b.txt"), []byte("b\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.Git.CommitAll(ctx, ws.Dir, "add b", false); err != nil {
		t.Fatal(err)
	}
	head, _ := m.Git.Rev(ctx, ws.Dir, "HEAD")
	if _, err := m.Git.Run(ctx, cache, "cat-file", "-e", head); err == nil {
		t.Fatal("workspace commit landed in the cache")
	}
	if got, _ := m.Git.Run(ctx, cache, "config", "--local", "--get", "user.name"); got != "" {
		t.Fatalf("workspace config landed in the cache: %q", got)
	}
	ws.Cleanup()

	// A later clone sees the moved base, fetched into the same cache.
	moved := remote.Commit("main", "moved", map[string]string{"c.txt": "c\n"})
	ws, err = m.Clone(ctx, "c", repo, "main")
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Cleanup()
	if got, _ := m.Git.Rev(ctx, ws.Dir, "HEAD"); got != moved || count(rec, "git fetch") != 2 || count(rec, "git init") != 3 {
		t.Fatalf("HEAD %s, want %s; calls %+v", got, moved, rec.Stats())
	}
	if _, err := os.Stat(filepath.Join(ws.Dir, "c.txt")); err != nil {
		t.Fatal(err)
	}
}

func TestCachedPrepareRebasesAndPushes(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	m, _ := cachedManager(t, remote, env)
	repo := ghapi.Repo{Owner: "acme", Name: "api"}

	// A new branch is pushed from a cached clone.
	ws, err := m.Prepare(ctx, "c", repo, "main", "fleet/c")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ws.Dir, "x.txt"), []byte("x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := m.Git.CommitAll(ctx, ws.Dir, "x", false); err != nil {
		t.Fatal(err)
	}
	if err := m.Git.Push(ctx, ws.Dir, "fleet/c", ""); err != nil {
		t.Fatal(err)
	}
	ws.Cleanup()

	// The existing branch is fetched in full and rebased onto the moved
	// base, through the same cache.
	remote.Commit("main", "moved", map[string]string{"b.txt": "b\n"})
	ws, err = m.Prepare(ctx, "c", repo, "main", "fleet/c")
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Cleanup()
	if ws.RemoteSHA != remote.Branch("fleet/c") || ws.Reset {
		t.Fatalf("existing: %+v", ws)
	}
	for _, f := range []string{"x.txt", "b.txt"} {
		if _, err := os.Stat(filepath.Join(ws.Dir, f)); err != nil {
			t.Fatalf("%s missing after rebase: %v", f, err)
		}
	}
	head, _ := m.Git.Rev(ctx, ws.Dir, "HEAD")
	if err := m.Git.Push(ctx, ws.Dir, "fleet/c", ws.RemoteSHA); err != nil || remote.Branch("fleet/c") != head {
		t.Fatalf("push: %v; branch %s, head %s", err, remote.Branch("fleet/c"), head)
	}
}

func TestCacheFallsBackToCloning(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	m, rec := cachedManager(t, remote, env)
	// A cache directory that is a file can't hold a cache.
	m.CacheDir = filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(m.CacheDir, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var failed []string
	m.OnCacheError = func(repo string, err error) { failed = append(failed, repo+": "+err.Error()) }
	ws, err := m.Clone(ctx, "c", ghapi.Repo{Owner: "acme", Name: "api"}, "main")
	if err != nil {
		t.Fatal(err)
	}
	defer ws.Cleanup()
	if len(failed) != 1 || !strings.HasPrefix(failed[0], "acme/api: update git cache:") || count(rec, "git clone") != 1 {
		t.Fatalf("failed %q, calls %+v", failed, rec.Stats())
	}
	if _, err := os.Stat(filepath.Join(ws.Dir, "a.txt")); err != nil {
		t.Fatal(err)
	}
	// A branch the remote doesn't have fails both ways.
	if _, err := m.Clone(ctx, "c", ghapi.Repo{Owner: "acme", Name: "api"}, "nope"); err == nil || len(failed) != 2 {
		t.Fatalf("missing branch: %v, %q", err, failed)
	}
}

func TestCacheFollowsURLAndCollectsGarbage(t *testing.T) {
	t.Parallel()
	ctx := t.Context()
	env := testutil.GitEnv(t)
	remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
	m, rec := cachedManager(t, remote, env)
	now := time.Now()
	m.Now = func() time.Time { return now }
	repo := ghapi.Repo{Owner: "acme", Name: "api"}
	cache := m.cachePath("acme", "api", remote.URL())
	stamp := filepath.Join(cache, "fleet-gc")

	clone := func() {
		t.Helper()
		ws, err := m.Clone(ctx, "c", repo, "main")
		if err != nil {
			t.Fatal(err)
		}
		ws.Cleanup()
	}
	// A new cache isn't collected, only stamped.
	clone()
	if _, err := os.Stat(stamp); err != nil || count(rec, "git gc") != 0 {
		t.Fatalf("stamp %v, gc %d", err, count(rec, "git gc"))
	}
	// Within a week, nothing is collected; after, the cache is.
	now = now.Add(cacheGCEvery - time.Minute)
	clone()
	if count(rec, "git gc") != 0 {
		t.Fatal("collected within a week")
	}
	now = now.Add(2 * time.Minute)
	clone()
	if st, _ := os.Stat(stamp); count(rec, "git gc") != 1 || st.ModTime().Sub(now).Abs() > time.Second {
		t.Fatalf("gc %d, stamp %v, now %v", count(rec, "git gc"), st.ModTime(), now)
	}

	// The cache's origin follows the clone URL.
	if _, err := m.Git.Run(ctx, cache, "config", "remote.origin.url", "file:///moved.git"); err != nil {
		t.Fatal(err)
	}
	clone()
	if got, _ := m.Git.Run(ctx, cache, "config", "--get", "remote.origin.url"); got != remote.URL() {
		t.Fatalf("cache origin = %q", got)
	}
	// Hosts don't share a cache.
	if m.cachePath("acme", "api", "https://a.example/acme/api.git") == m.cachePath("acme", "api", "https://b.example/acme/api.git") {
		t.Fatal("two hosts share a cache")
	}
}

func TestCacheGitFailuresFallBackToCloning(t *testing.T) {
	t.Parallel()
	// Each git step of the cache failing in turn: in a new cache, and in
	// one an earlier run filled.
	for _, tc := range []struct {
		name string
		fail string
		warm bool
	}{
		{name: "create the cache", fail: "init"},
		{name: "turn off automatic gc", fail: "config:0"},
		{name: "point the cache at the remote", fail: "config:URL"},
		{name: "fetch into the cache", fail: "fetch"},
		{name: "create the workspace", fail: "init", warm: true},
		{name: "point the workspace at the cache's commit", fail: "update-ref", warm: true},
		{name: "check out the base", fail: "checkout:origin/main", warm: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ctx := t.Context()
			env := testutil.GitEnv(t)
			remote := testutil.NewRemote(t, env, map[string]string{"a.txt": "a\n"})
			cacheDir := t.TempDir()
			repo := ghapi.Repo{Owner: "acme", Name: "api"}
			if tc.warm {
				warm := &Manager{Root: t.TempDir(), Git: testutil.Git(env), CacheDir: cacheDir, CloneURL: func(ghapi.Repo) string { return remote.URL() }}
				ws, err := warm.Clone(ctx, "c", repo, "main")
				if err != nil {
					t.Fatal(err)
				}
				ws.Cleanup()
			}
			fail := strings.Replace(tc.fail, "URL", remote.URL(), 1)
			m := &Manager{
				Root: t.TempDir(), CacheDir: cacheDir,
				Git:      testutil.Git(slices.Concat(env, []string{failGitEnv + "=" + fail})),
				CloneURL: func(ghapi.Repo) string { return remote.URL() },
			}
			m.Git.Bin = os.Args[0]
			var failed []error
			m.OnCacheError = func(_ string, err error) { failed = append(failed, err) }
			ws, err := m.Clone(ctx, "c", repo, "main")
			if err != nil {
				t.Fatal(err)
			}
			defer ws.Cleanup()
			if len(failed) != 1 || !strings.Contains(failed[0].Error(), "injected failure") {
				t.Fatalf("cache errors = %v", failed)
			}
			// The direct clone made a complete workspace of its own.
			if b, err := os.ReadFile(filepath.Join(ws.Dir, "a.txt")); err != nil || string(b) != "a\n" {
				t.Fatalf("a.txt = %q, %v", b, err)
			}
			if _, err := os.Stat(filepath.Join(ws.Dir, ".git", "objects", "info", "alternates")); err == nil {
				t.Fatal("fallback clone borrows from the cache")
			}
		})
	}
}
