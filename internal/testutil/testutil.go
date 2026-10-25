// Package testutil builds git fixtures for tests: an isolated git config
// and local bare repositories created in t.TempDir(), served over file://
// so shallow and partial clones behave like they do against GitHub.
package testutil

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rossedman/gh-fleet/internal/gitops"
)

// GitEnv returns environment variables that isolate git from the user's
// config, with a fixed identity, and writes the config file they point at.
func GitEnv(t testing.TB) []string {
	t.Helper()
	dir := t.TempDir()
	cfg := filepath.Join(dir, "gitconfig")
	content := "[user]\n\tname = Fleet Test\n\temail = fleet@example.com\n" +
		"[init]\n\tdefaultBranch = main\n" +
		"[uploadpack]\n\tallowFilter = true\n\tallowAnySHA1InWant = true\n" +
		"[protocol \"file\"]\n\tallow = always\n" +
		"[commit]\n\tgpgsign = false\n"
	if err := os.WriteFile(cfg, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return []string{
		"GIT_CONFIG_GLOBAL=" + cfg,
		"GIT_CONFIG_NOSYSTEM=1",
		"HOME=" + dir,
	}
}

// Git returns a gitops.Git using env.
func Git(env []string) gitops.Git {
	return gitops.Git{Env: env}
}

// run runs git in dir and fails the test on error.
func run(t testing.TB, env []string, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(string(out))
}

// Remote is a bare repository with a scratch clone for making commits.
type Remote struct {
	t    testing.TB
	env  []string
	Bare string
	work string
}

// URL returns the file:// URL of the bare repo.
func (r *Remote) URL() string { return "file://" + r.Bare }

// NewRemote creates a bare repo whose main branch holds files.
func NewRemote(t testing.TB, env []string, files map[string]string) *Remote {
	t.Helper()
	root := t.TempDir()
	r := &Remote{t: t, env: env, Bare: filepath.Join(root, "remote.git"), work: filepath.Join(root, "work")}
	run(t, env, root, "init", "--quiet", "--bare", "--initial-branch=main", r.Bare)
	run(t, env, root, "clone", "--quiet", r.Bare, r.work)
	r.Commit("main", "initial commit", files)
	return r
}

// Commit writes files on branch (creating it from main if needed) and
// pushes the commit. A value of "" deletes the file.
func (r *Remote) Commit(branch, msg string, files map[string]string) string {
	r.t.Helper()
	if run(r.t, r.env, r.work, "ls-remote", "--heads", "origin", "main") != "" {
		run(r.t, r.env, r.work, "fetch", "--quiet", "origin")
		start := "origin/main"
		if run(r.t, r.env, r.work, "ls-remote", "--heads", "origin", branch) != "" {
			start = "origin/" + branch
		}
		run(r.t, r.env, r.work, "checkout", "--quiet", "-B", branch, start)
	}
	for name, content := range files {
		p := filepath.Join(r.work, name)
		if content == "" {
			_ = os.Remove(p)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			r.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			r.t.Fatal(err)
		}
	}
	run(r.t, r.env, r.work, "add", "--all")
	run(r.t, r.env, r.work, "commit", "--quiet", "--allow-empty", "-m", msg)
	run(r.t, r.env, r.work, "push", "--quiet", "--force", "origin", "HEAD:refs/heads/"+branch)
	return run(r.t, r.env, r.work, "rev-parse", "HEAD")
}

// Branch returns the SHA of branch in the bare repo, or "".
func (r *Remote) Branch(branch string) string {
	r.t.Helper()
	out := run(r.t, r.env, r.Bare, "for-each-ref", "--format=%(objectname)", "refs/heads/"+branch)
	return out
}

// Show returns the content of path at rev in the bare repo.
func (r *Remote) Show(rev, path string) string {
	r.t.Helper()
	return run(r.t, r.env, r.Bare, "show", rev+":"+path)
}

// Log returns the commit subjects of rev, newest first.
func (r *Remote) Log(rev string) []string {
	r.t.Helper()
	return strings.Split(run(r.t, r.env, r.Bare, "log", "--format=%s", rev), "\n")
}
