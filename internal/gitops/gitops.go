// Package gitops runs the git operations gh fleet needs by shelling out to
// the user's git, so credential helpers, signing config and proxies behave
// exactly as they do for the user.
package gitops

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Git runs git commands.
type Git struct {
	// Bin is the git executable; defaults to "git".
	Bin string
	// Env is added to the environment of every git command.
	Env []string
}

// bin returns the git executable.
func (g Git) bin() string {
	if g.Bin != "" {
		return g.Bin
	}
	return "git"
}

// Run runs git in dir and returns trimmed stdout. Errors include stderr.
func (g Git) Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, g.bin(), args...)
	cmd.Dir = dir
	cmd.Env = append(append(os.Environ(), "GIT_TERMINAL_PROMPT=0"), g.Env...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", args[0], msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

// Clone makes a shallow, blobless, single-branch clone of branch into dir.
// Servers that do not support partial clone fall back to a full blob fetch.
func (g Git) Clone(ctx context.Context, url, dir, branch string) error {
	_, err := g.Run(ctx, "", "clone", "--quiet", "--depth", "1", "--filter=blob:none", "--single-branch", "--branch", branch, url, dir)
	return err
}

// RemoteBranchSHA returns the SHA of branch on origin, or "" when the
// branch does not exist.
func (g Git) RemoteBranchSHA(ctx context.Context, dir, branch string) (string, error) {
	out, err := g.Run(ctx, dir, "ls-remote", "--heads", "origin", "refs/heads/"+branch)
	if err != nil {
		return "", err
	}
	sha, _, _ := strings.Cut(out, "\t")
	return sha, nil
}

// FetchBranches fetches full history (without blobs) for the given
// branches, deepening a shallow clone so they can be rebased.
func (g Git) FetchBranches(ctx context.Context, dir string, branches ...string) error {
	args := []string{"fetch", "--quiet", "--filter=blob:none"}
	if shallow, err := g.Run(ctx, dir, "rev-parse", "--is-shallow-repository"); err == nil && shallow == "true" {
		args = append(args, "--unshallow")
	}
	args = append(args, "origin")
	for _, b := range branches {
		args = append(args, "+refs/heads/"+b+":refs/remotes/origin/"+b)
	}
	_, err := g.Run(ctx, dir, args...)
	return err
}

// Checkout creates or resets branch to start and checks it out.
func (g Git) Checkout(ctx context.Context, dir, branch, start string) error {
	_, err := g.Run(ctx, dir, "checkout", "--quiet", "-B", branch, start)
	return err
}

// Rebase rebases the current branch onto upstream. On conflict it aborts
// the rebase and returns an error, leaving the branch as it was.
func (g Git) Rebase(ctx context.Context, dir, upstream string) error {
	if _, err := g.Run(ctx, dir, "rebase", "--quiet", upstream); err != nil {
		_, _ = g.Run(context.WithoutCancel(ctx), dir, "rebase", "--abort")
		return err
	}
	return nil
}

// Dirty reports whether the work tree has changes, untracked files
// included, using git status --porcelain (which respects .gitignore).
func (g Git) Dirty(ctx context.Context, dir string) (bool, error) {
	out, err := g.Run(ctx, dir, "status", "--porcelain")
	if err != nil {
		return false, err
	}
	return out != "", nil
}

// StagedDiff stages every change and returns the unified diff between
// base and the index, which is what the PR will contain.
func (g Git) StagedDiff(ctx context.Context, dir, base string) (string, error) {
	if _, err := g.Run(ctx, dir, "add", "--all"); err != nil {
		return "", err
	}
	cmd := []string{"-c", "color.ui=never", "diff", "--cached", "--no-ext-diff", base}
	return g.Run(ctx, dir, cmd...)
}

// CommitAll stages every change and commits it with message, author and
// signing taken from the user's git config.
func (g Git) CommitAll(ctx context.Context, dir, message string, signoff bool) error {
	if _, err := g.Run(ctx, dir, "add", "--all"); err != nil {
		return err
	}
	args := []string{"commit", "--quiet", "--no-verify", "-m", message}
	if signoff {
		args = append(args, "--signoff")
	}
	_, err := g.Run(ctx, dir, args...)
	return err
}

// Rev resolves a revision to a SHA.
func (g Git) Rev(ctx context.Context, dir, rev string) (string, error) {
	return g.Run(ctx, dir, "rev-parse", "--verify", "--quiet", rev+"^{commit}")
}

// SameTree reports whether two revisions have identical content.
func (g Git) SameTree(ctx context.Context, dir, a, b string) (bool, error) {
	ta, err := g.Run(ctx, dir, "rev-parse", a+"^{tree}")
	if err != nil {
		return false, err
	}
	tb, err := g.Run(ctx, dir, "rev-parse", b+"^{tree}")
	if err != nil {
		return false, err
	}
	return ta == tb, nil
}

// Push force-pushes HEAD to branch on origin, but only if the remote
// branch is still at lease ("" means it must not exist yet).
func (g Git) Push(ctx context.Context, dir, branch, lease string) error {
	ref := "refs/heads/" + branch
	_, err := g.Run(ctx, dir, "push", "--quiet", "--force-with-lease="+ref+":"+lease, "origin", "HEAD:"+ref)
	return err
}
