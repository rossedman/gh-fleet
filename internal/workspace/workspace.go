// Package workspace prepares a fresh clone of a target repo for a change:
// a shallow clone of the base branch under
// $TMPDIR/gh-fleet/<campaign>/<owner>/<repo>, with the campaign branch
// checked out and, if it already existed, rebased onto the base.
package workspace

import (
	"context"
	"os"
	"path/filepath"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/gitops"
)

// Manager creates and removes workspaces.
type Manager struct {
	// Root is the parent of every workspace; defaults to $TMPDIR/gh-fleet.
	Root string
	Git  gitops.Git
	// CloneURL returns the URL to clone a repo from; defaults to the
	// repo's https URL with .git appended.
	CloneURL func(ghapi.Repo) string
	// Keep leaves workspaces on disk after Cleanup, for debugging.
	Keep bool
}

// Workspace is one prepared clone.
type Workspace struct {
	Dir    string
	Base   string
	Branch string
	// RemoteSHA is the campaign branch's SHA on origin, or "" if new.
	RemoteSHA string
	// Reset is true when an existing branch could not be rebased and was
	// reset to the base branch instead.
	Reset bool

	keep bool
}

// DefaultRoot returns $TMPDIR/gh-fleet.
func DefaultRoot() string {
	return filepath.Join(os.TempDir(), "gh-fleet")
}

// Path returns where a repo's workspace lives.
func (m *Manager) Path(campaign string, repo ghapi.Repo) string {
	root := m.Root
	if root == "" {
		root = DefaultRoot()
	}
	return filepath.Join(root, campaign, filepath.Base(repo.Owner), filepath.Base(repo.Name))
}

// url returns the clone URL for repo.
func (m *Manager) url(repo ghapi.Repo) string {
	if m.CloneURL != nil {
		return m.CloneURL(repo)
	}
	u := repo.URL
	if u == "" {
		u = "https://github.com/" + repo.FullName()
	}
	return u + ".git"
}

// Prepare clones repo's base branch and checks out branch. An existing
// remote branch is rebased onto base; on conflict it is reset to base, as
// the script, not the old branch, is the source of truth. It asks origin
// where branch is; PrepareAt takes that from the caller instead.
func (m *Manager) Prepare(ctx context.Context, campaign string, repo ghapi.Repo, base, branch string) (*Workspace, error) {
	return m.prepare(ctx, campaign, repo, base, branch, nil)
}

// PrepareAt is Prepare for a branch the caller already looked up: remote
// is its commit on origin, or "" when it doesn't exist. Looking branches
// up in batches saves a round trip to the git server per repo. A branch
// that moved since is caught by the push's lease.
func (m *Manager) PrepareAt(ctx context.Context, campaign string, repo ghapi.Repo, base, branch, remote string) (*Workspace, error) {
	return m.prepare(ctx, campaign, repo, base, branch, &remote)
}

// prepare is Prepare, or PrepareAt when remote is set.
func (m *Manager) prepare(ctx context.Context, campaign string, repo ghapi.Repo, base, branch string, remote *string) (*Workspace, error) {
	ws, err := m.Clone(ctx, campaign, repo, base)
	if err != nil {
		return nil, err
	}
	ws.Branch = branch
	var sha string
	if remote != nil {
		sha = *remote
	} else if sha, err = m.Git.RemoteBranchSHA(ctx, ws.Dir, branch); err != nil {
		ws.Cleanup()
		return nil, err
	}
	dir := ws.Dir
	ws.RemoteSHA = sha
	if sha == "" {
		if err := m.Git.Checkout(ctx, dir, branch, "origin/"+base); err != nil {
			ws.Cleanup()
			return nil, err
		}
		return ws, nil
	}
	if err := m.Git.FetchBranches(ctx, dir, base, branch); err != nil {
		ws.Cleanup()
		return nil, err
	}
	if err := m.Git.Checkout(ctx, dir, branch, "origin/"+branch); err != nil {
		ws.Cleanup()
		return nil, err
	}
	if err := m.Git.Rebase(ctx, dir, "origin/"+base); err != nil {
		if err := m.Git.Checkout(ctx, dir, branch, "origin/"+base); err != nil {
			ws.Cleanup()
			return nil, err
		}
		ws.Reset = true
	}
	return ws, nil
}

// Clone makes a read-only workspace: a shallow clone of base with no
// campaign branch, for commands that only inspect repos.
func (m *Manager) Clone(ctx context.Context, campaign string, repo ghapi.Repo, base string) (*Workspace, error) {
	dir := m.Path(campaign, repo)
	if err := os.RemoveAll(dir); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(dir), 0o700); err != nil {
		return nil, err
	}
	ws := &Workspace{Dir: dir, Base: base, keep: m.Keep}
	if err := m.Git.Clone(ctx, m.url(repo), dir, base); err != nil {
		ws.Cleanup()
		return nil, err
	}
	return ws, nil
}

// Cleanup removes the workspace unless it is kept for debugging.
func (w *Workspace) Cleanup() {
	if w.keep {
		return
	}
	_ = os.RemoveAll(w.Dir)
}
