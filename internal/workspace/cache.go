package workspace

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"time"
)

// cacheGCEvery is how often a cached repo is garbage collected, dropping
// the files of base commits that later fetches replaced.
const cacheGCEvery = 7 * 24 * time.Hour

// cachePath returns where repo's cache lives: one bare repo per clone URL,
// named for the repo and a hash of the URL so hosts don't collide.
func (m *Manager) cachePath(owner, name, url string) string {
	sum := sha256.Sum256([]byte(url))
	return filepath.Join(m.CacheDir, filepath.Base(owner), filepath.Base(name)+"-"+hex.EncodeToString(sum[:4])+".git")
}

// cachedClone makes dir a clone of url's branch, as Git.Clone does, but
// with the files from a cache kept between runs. The cache is a shallow
// bare repo per URL that each run brings up to date with a fetch, which
// transfers only what changed since the last one. The workspace is a
// repo of its own, with origin pointing at url, that borrows the cache's
// objects through objects/info/alternates instead of copying them; what
// a script or a commit writes goes into the workspace, not the cache.
func (m *Manager) cachedClone(ctx context.Context, owner, name, url, dir, branch string) error {
	cache := m.cachePath(owner, name, url)
	sha, err := m.updateCache(ctx, cache, url, branch)
	if err != nil {
		return fmt.Errorf("update git cache: %w", err)
	}
	if _, err := m.Git.Run(ctx, "", "init", "--quiet", dir); err != nil {
		return err
	}
	gitDir := filepath.Join(dir, ".git")
	objects := filepath.ToSlash(filepath.Join(cache, "objects"))
	if err := os.WriteFile(filepath.Join(gitDir, "objects", "info", "alternates"), []byte(objects+"\n"), 0o600); err != nil {
		return err
	}
	// The cache's history is shallow, and so is the workspace that
	// borrows it.
	switch b, err := os.ReadFile(filepath.Join(cache, "shallow")); {
	case err == nil:
		if err := os.WriteFile(filepath.Join(gitDir, "shallow"), b, 0o600); err != nil {
			return err
		}
	case !errors.Is(err, fs.ErrNotExist):
		return err
	}
	for _, args := range [][]string{
		{"config", "remote.origin.url", url},
		{"config", "remote.origin.fetch", "+refs/heads/" + branch + ":refs/remotes/origin/" + branch},
		{"update-ref", "refs/remotes/origin/" + branch, sha},
		{"checkout", "--quiet", "-B", branch, "--track", "origin/" + branch},
	} {
		if _, err := m.Git.Run(ctx, dir, args...); err != nil {
			return err
		}
	}
	return nil
}

// updateCache creates the cache for url when it doesn't exist yet, fetches
// branch into it and returns the commit branch is at. Two runs fetching
// into one cache at once can fail on git's locks; the caller then clones
// without the cache.
func (m *Manager) updateCache(ctx context.Context, cache, url, branch string) (string, error) {
	if _, err := os.Stat(filepath.Join(cache, "HEAD")); err != nil {
		if err := os.MkdirAll(filepath.Dir(cache), 0o700); err != nil {
			return "", err
		}
		if _, err := m.Git.Run(ctx, "", "init", "--quiet", "--bare", cache); err != nil {
			return "", err
		}
		// gh fleet collects garbage itself, between fetches, so git never
		// deletes objects a workspace may be borrowing mid-run.
		if _, err := m.Git.Run(ctx, cache, "config", "gc.auto", "0"); err != nil {
			return "", err
		}
	}
	if cur, _ := m.Git.Run(ctx, cache, "config", "--get", "remote.origin.url"); cur != url {
		if _, err := m.Git.Run(ctx, cache, "config", "remote.origin.url", url); err != nil {
			return "", err
		}
	}
	ref := "refs/remotes/origin/" + branch
	if _, err := m.Git.Run(ctx, cache, "fetch", "--quiet", "--no-tags", "--depth", "1", "origin", "+refs/heads/"+branch+":"+ref); err != nil {
		return "", err
	}
	m.collectGarbage(ctx, cache)
	return m.Git.Rev(ctx, cache, ref)
}

// collectGarbage runs git gc in the cache once every cacheGCEvery, so the
// files of base commits that later fetches replaced don't pile up. gc only
// prunes unreachable objects written more than two weeks ago, and an
// object becomes unreachable only when a fetch moves the base past it, so
// a run can lose objects it borrows only when another run fetches into
// the same cache and collects garbage while it works; that repo fails and
// retry clones it again. A failure leaves the cache as it was.
func (m *Manager) collectGarbage(ctx context.Context, cache string) {
	stamp := filepath.Join(cache, "fleet-gc")
	st, err := os.Stat(stamp)
	if err == nil && m.now().Sub(st.ModTime()) < cacheGCEvery {
		return
	}
	if errors.Is(err, fs.ErrNotExist) {
		// A new cache has nothing to collect yet.
		_ = os.WriteFile(stamp, nil, 0o600)
		return
	}
	if _, err := m.Git.Run(ctx, cache, "gc", "--quiet"); err == nil {
		now := m.now()
		_ = os.Chtimes(stamp, now, now)
	}
}

// now returns the manager's clock.
func (m *Manager) now() time.Time {
	if m.Now != nil {
		return m.Now()
	}
	return time.Now()
}
