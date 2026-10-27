// Package state stores local campaign state.
//
// GitHub is the source of truth for pull requests; local state is a cache
// plus error history. Each campaign is one JSON file under
// $XDG_STATE_HOME/gh-fleet/<name>.json, and per-repo script logs live in
// $XDG_STATE_HOME/gh-fleet/<name>/logs/.
package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// Status is where a repo is in a campaign.
type Status string

// Repo statuses.
const (
	Pending Status = "pending"
	Noop    Status = "noop"
	Changed Status = "changed"
	PROpen  Status = "pr_open"
	Merged  Status = "merged"
	Closed  Status = "closed"
	Failed  Status = "failed"
	Skipped Status = "skipped"
	// Done marks a repo where gh fleet exec ran successfully.
	Done Status = "done"
)

// RepoState is everything recorded about one repo in a campaign.
type RepoState struct {
	Repo     string `json:"repo"`
	Status   Status `json:"status"`
	PRNumber int    `json:"prNumber,omitempty"`
	PRURL    string `json:"prUrl,omitempty"`
	HeadSHA  string `json:"headSha,omitempty"`
	Checks   string `json:"checks,omitempty"`
	Review   string `json:"review,omitempty"`
	// MergeState is the PR's mergeStateStatus, such as CLEAN or DIRTY.
	MergeState string `json:"mergeState,omitempty"`
	LastError  string `json:"lastError,omitempty"`
	Detail     string `json:"detail,omitempty"`
	Diff       string `json:"diff,omitempty"`
	Output     string `json:"output,omitempty"`
	// BaseSHA and Fingerprint describe a run that found nothing to change:
	// the base commit it ran on, and the change's fingerprint for the repo.
	BaseSHA     string    `json:"baseSha,omitempty"`
	Fingerprint string    `json:"fingerprint,omitempty"`
	CreatedAt   time.Time `json:"createdAt,omitzero"`
	UpdatedAt   time.Time `json:"updatedAt,omitzero"`
}

// Campaign is the stored state of one campaign.
type Campaign struct {
	Name      string                `json:"name"`
	File      string                `json:"file,omitempty"`
	UpdatedAt time.Time             `json:"updatedAt,omitzero"`
	Repos     map[string]*RepoState `json:"repos"`
	// Unchanged records, per repo, the last plan or apply that found
	// nothing to change, so apply can skip the repo while neither its base
	// nor the change has moved.
	Unchanged map[string]Unchanged `json:"unchanged,omitempty"`

	mu sync.Mutex
}

// Get returns the state for repo, or nil.
func (c *Campaign) Get(repo string) *RepoState {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Repos[repo]
}

// Set records a repo's state, keeping its creation time. Diffs and command
// output are not stored.
func (c *Campaign) Set(s RepoState, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Repos == nil {
		c.Repos = map[string]*RepoState{}
	}
	if old, ok := c.Repos[s.Repo]; ok {
		s.CreatedAt = old.CreatedAt
	}
	if s.CreatedAt.IsZero() {
		s.CreatedAt = now
	}
	s.UpdatedAt = now
	s.Diff, s.Output = "", ""
	c.Repos[s.Repo] = &s
	c.UpdatedAt = now
}

// Unchanged is a run that found nothing to change in a repo.
type Unchanged struct {
	BaseSHA     string    `json:"baseSha"`
	Fingerprint string    `json:"fingerprint"`
	At          time.Time `json:"at"`
}

// NoteResult records whether a run left a repo unchanged: a no-op with a
// fingerprint is remembered, anything else forgets the repo.
func (c *Campaign) NoteResult(s RepoState, now time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if s.Status != Noop || s.Fingerprint == "" || s.BaseSHA == "" {
		delete(c.Unchanged, s.Repo)
		return
	}
	if c.Unchanged == nil {
		c.Unchanged = map[string]Unchanged{}
	}
	c.Unchanged[s.Repo] = Unchanged{BaseSHA: s.BaseSHA, Fingerprint: s.Fingerprint, At: now}
}

// UnchangedFor returns the last run that left repo unchanged.
func (c *Campaign) UnchangedFor(repo string) (Unchanged, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	u, ok := c.Unchanged[repo]
	return u, ok
}

// Sorted returns repo states ordered by repo name.
func (c *Campaign) Sorted() []RepoState {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]RepoState, 0, len(c.Repos))
	for _, r := range c.Repos {
		out = append(out, *r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Repo < out[j].Repo })
	return out
}

// Store reads and writes campaign state files.
type Store struct {
	Dir string
	// Now returns the current time; defaults to time.Now.
	Now func() time.Time
}

// DefaultDir returns $XDG_STATE_HOME/gh-fleet, falling back to
// ~/.local/state/gh-fleet.
func DefaultDir() (string, error) {
	if d := os.Getenv("XDG_STATE_HOME"); d != "" {
		return filepath.Join(d, "gh-fleet"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state", "gh-fleet"), nil
}

// Time returns the store's current time.
func (s *Store) Time() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// path returns the state file for a campaign.
func (s *Store) path(name string) string {
	return filepath.Join(s.Dir, name+".json")
}

// LogDir returns the directory for a campaign's per-repo logs.
func (s *Store) LogDir(name string) string {
	return filepath.Join(s.Dir, name, "logs")
}

// LogPath returns the log file for one repo in a campaign.
func (s *Store) LogPath(name, repo string) string {
	return filepath.Join(s.LogDir(name), strings.ReplaceAll(repo, "/", "__")+".log")
}

// Load returns a campaign's state, or an empty state when none is stored.
func (s *Store) Load(name string) (*Campaign, error) {
	b, err := os.ReadFile(s.path(name))
	if errors.Is(err, fs.ErrNotExist) {
		return &Campaign{Name: name, Repos: map[string]*RepoState{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var c Campaign
	if err := json.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("read state for %s: %w", name, err)
	}
	if c.Repos == nil {
		c.Repos = map[string]*RepoState{}
	}
	c.Name = name
	return &c, nil
}

// Exists reports whether a campaign has stored state.
func (s *Store) Exists(name string) bool {
	_, err := os.Stat(s.path(name))
	return err == nil
}

// Save writes a campaign's state atomically.
func (s *Store) Save(c *Campaign) error {
	c.mu.Lock()
	b, err := json.MarshalIndent(c, "", "  ")
	c.mu.Unlock()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(s.Dir, 0o700); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(s.Dir, c.Name+".*.tmp")
	if err != nil {
		return err
	}
	if _, err := tmp.Write(append(b, '\n')); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return os.Rename(tmp.Name(), s.path(c.Name))
}

// Summary is one campaign in a listing.
type Summary struct {
	Name      string         `json:"name"`
	File      string         `json:"file,omitempty"`
	UpdatedAt time.Time      `json:"updatedAt,omitzero"`
	Counts    map[Status]int `json:"counts"`
}

// List summarizes every stored campaign, ordered by name.
func (s *Store) List() ([]Summary, error) {
	entries, err := os.ReadDir(s.Dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []Summary
	for _, e := range entries {
		name, ok := strings.CutSuffix(e.Name(), ".json")
		if e.IsDir() || !ok {
			continue
		}
		c, err := s.Load(name)
		if err != nil {
			return nil, err
		}
		sum := Summary{Name: name, File: c.File, UpdatedAt: c.UpdatedAt, Counts: map[Status]int{}}
		for _, r := range c.Repos {
			sum.Counts[r.Status]++
		}
		out = append(out, sum)
	}
	return out, nil
}
