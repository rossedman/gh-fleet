// Package pr finds, creates, updates and closes a campaign's pull
// requests, idempotently.
//
// A campaign's PRs are found by head branch first, in one batched query
// per 50 repos, then by the hidden marker in their body via search, so
// they are found even when local state is gone.
package pr

import (
	"context"
	"slices"
	"sync"

	"github.com/rossedman/gh-fleet/internal/ghapi"
)

// Action is what Sync did.
type Action string

// Sync actions.
const (
	Created      Action = "created"
	Updated      Action = "updated"
	Unchanged    Action = "unchanged"
	Reopened     Action = "reopened"
	LimitReached Action = "limit_reached"
)

// Desired is the PR a campaign wants in one repo.
type Desired struct {
	Repo      string
	Title     string
	Body      string
	Head      string
	Base      string
	Draft     bool
	Labels    []string
	Reviewers []string
}

// Manager tracks a campaign's PRs across one run.
type Manager struct {
	Client ghapi.Client
	// MaxOpen caps open campaign PRs; 0 means no limit.
	MaxOpen int

	mu    sync.Mutex
	index map[string]ghapi.PR
	open  int
}

// NewManager returns a Manager for client.
func NewManager(client ghapi.Client, maxOpen int) *Manager {
	return &Manager{Client: client, MaxOpen: maxOpen, index: map[string]ghapi.PR{}}
}

// Index loads the existing PRs for repos: by head branch in batched
// queries, then by marker search scoped with qualifiers for repos whose PR
// was not found by branch. It also counts open campaign PRs.
func (m *Manager) Index(ctx context.Context, repos []string, head, marker string, qualifiers []string) error {
	byHead, err := m.Client.PRsByHead(ctx, repos, head)
	if err != nil {
		return err
	}
	found := byHead
	if found == nil {
		found = map[string]ghapi.PR{}
	}
	if len(found) < len(repos) && marker != "" {
		hits, err := m.Client.SearchCampaignPRs(ctx, qualifiers, marker)
		if err != nil {
			return err
		}
		want := map[string]bool{}
		for _, r := range repos {
			want[r] = true
		}
		for _, p := range hits {
			if _, ok := found[p.Repo]; !ok && want[p.Repo] {
				found[p.Repo] = p
			}
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.index = found
	m.open = 0
	for _, p := range found {
		if p.State == ghapi.StateOpen {
			m.open++
		}
	}
	return nil
}

// Existing returns the indexed PR for repo, or nil.
func (m *Manager) Existing(repo string) *ghapi.PR {
	m.mu.Lock()
	defer m.mu.Unlock()
	p, ok := m.index[repo]
	if !ok {
		return nil
	}
	return &p
}

// All returns every indexed PR.
func (m *Manager) All() []ghapi.PR {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]ghapi.PR, 0, len(m.index))
	for _, p := range m.index {
		out = append(out, p)
	}
	return out
}

// OpenCount returns how many campaign PRs are open.
func (m *Manager) OpenCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.open
}

// Reserve claims a slot for one more open PR. It returns false when
// MaxOpen is reached. Call Release if the PR is not opened after all.
func (m *Manager) Reserve() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.MaxOpen > 0 && m.open >= m.MaxOpen {
		return false
	}
	m.open++
	return true
}

// Release gives back a slot claimed with Reserve.
func (m *Manager) Release() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.open--
}

// remember stores the latest view of a PR.
func (m *Manager) remember(p ghapi.PR) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.index[p.Repo] = p
}

// Sync makes the repo's PR match d. With no existing PR it creates one
// (the caller must have reserved a slot); an open PR has its title, body
// and labels updated when they differ; a closed PR is reopened only when
// reopen is true (the caller must have reserved a slot for that too), or
// replaced by a new PR when GitHub refuses to reopen it.
func (m *Manager) Sync(ctx context.Context, d Desired, existing *ghapi.PR, reopen bool) (ghapi.PR, Action, error) {
	if existing == nil {
		p, err := m.Client.CreatePR(ctx, d.Repo, ghapi.NewPR{Title: d.Title, Body: d.Body, Head: d.Head, Base: d.Base, Draft: d.Draft})
		if err != nil {
			return ghapi.PR{}, "", err
		}
		if err := m.Client.AddLabels(ctx, d.Repo, p.Number, d.Labels); err != nil {
			return p, Created, err
		}
		p.Labels = d.Labels
		if err := m.Client.RequestReviewers(ctx, d.Repo, p.Number, d.Reviewers); err != nil {
			return p, Created, err
		}
		m.remember(p)
		return p, Created, nil
	}

	p := *existing
	action := Unchanged
	var u ghapi.PRUpdate
	if p.State == ghapi.StateClosed && reopen {
		open := "open"
		u.State = &open
		action = Reopened
	}
	if p.Title != d.Title {
		u.Title = &d.Title
	}
	if p.Body != d.Body {
		u.Body = &d.Body
	}
	if u.Title != nil || u.Body != nil || u.State != nil {
		if err := m.Client.UpdatePR(ctx, d.Repo, p.Number, u); err != nil {
			if action == Reopened {
				// GitHub refuses to reopen a PR whose branch was
				// force-pushed or recreated; open a fresh one instead.
				return m.Sync(ctx, d, nil, false)
			}
			return p, "", err
		}
		p.Title, p.Body = d.Title, d.Body
		if action == Unchanged {
			action = Updated
		}
	}
	if action == Reopened {
		p.State = ghapi.StateOpen
	}
	var missing []string
	for _, l := range d.Labels {
		if !slices.Contains(p.Labels, l) {
			missing = append(missing, l)
		}
	}
	if len(missing) > 0 {
		if err := m.Client.AddLabels(ctx, d.Repo, p.Number, missing); err != nil {
			return p, "", err
		}
		p.Labels = append(p.Labels, missing...)
		if action == Unchanged {
			action = Updated
		}
	}
	m.remember(p)
	return p, action, nil
}

// Close comments on (when comment is non-empty) and closes a PR, then
// deletes its branch when deleteBranch is true.
func (m *Manager) Close(ctx context.Context, p ghapi.PR, comment string, deleteBranch bool) error {
	if comment != "" {
		if err := m.Client.Comment(ctx, p.Repo, p.Number, comment); err != nil {
			return err
		}
	}
	closed := "closed"
	if err := m.Client.UpdatePR(ctx, p.Repo, p.Number, ghapi.PRUpdate{State: &closed}); err != nil {
		return err
	}
	if deleteBranch && p.HeadRef != "" {
		if err := m.Client.DeleteBranch(ctx, p.Repo, p.HeadRef); err != nil {
			return err
		}
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if old, ok := m.index[p.Repo]; ok && old.State == ghapi.StateOpen {
		m.open--
	}
	p.State = ghapi.StateClosed
	m.index[p.Repo] = p
	return nil
}
