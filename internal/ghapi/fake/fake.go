// Package fake is an in-memory ghapi.Client for tests and benchmarks.
//
// It keeps repos, files, code search hits and pull requests in memory,
// records every write, and lets a test inject an error per method.
package fake

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/rossedman/gh-fleet/internal/ghapi"
)

// Client is an in-memory GitHub.
type Client struct {
	mu sync.Mutex

	// Repos by full name.
	Repos map[string]ghapi.Repo
	// Files lists the paths that exist per repo full name.
	Files map[string][]string
	// Search maps a code search query to the repos it hits.
	Search map[string][]string
	// PRs holds every pull request, in creation order.
	PRs []*ghapi.PR
	// Errors makes the named method (e.g. "CreatePR") fail. A key of
	// "CreatePR:owner/name" fails only for that repo.
	Errors map[string]error
	// Limits is returned by RateLimits.
	Limits []ghapi.RateLimit

	// Calls counts calls per method name.
	Calls map[string]int
	// Comments records comment bodies per "repo#number".
	Comments map[string][]string
	// Reviewers records requested reviewers per "repo#number".
	Reviewers map[string][]string
	// DeletedBranches records "repo:branch" for each deleted branch.
	DeletedBranches []string
	// Delay is added to every call, to simulate latency in benchmarks.
	Delay time.Duration
}

// New returns an empty fake.
func New() *Client {
	return &Client{
		Repos:     map[string]ghapi.Repo{},
		Files:     map[string][]string{},
		Search:    map[string][]string{},
		Errors:    map[string]error{},
		Calls:     map[string]int{},
		Comments:  map[string][]string{},
		Reviewers: map[string][]string{},
	}
}

// AddRepo adds a repo, defaulting its branch to main and push access on.
func (c *Client) AddRepo(r ghapi.Repo) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if r.DefaultBranch == "" {
		r.DefaultBranch = "main"
	}
	c.Repos[r.FullName()] = r
}

// call records a call and returns the injected error for it, if any.
func (c *Client) call(ctx context.Context, method, repo string) error {
	c.Calls[method]++
	if c.Delay > 0 {
		c.mu.Unlock()
		time.Sleep(c.Delay)
		c.mu.Lock()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err, ok := c.Errors[method+":"+repo]; ok {
		return err
	}
	return c.Errors[method]
}

// ListOwnerRepos returns the repos owned by owner, sorted by name.
func (c *Client) ListOwnerRepos(ctx context.Context, owner string) ([]ghapi.Repo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "ListOwnerRepos", owner); err != nil {
		return nil, err
	}
	var out []ghapi.Repo
	for _, r := range c.Repos {
		if strings.EqualFold(r.Owner, owner) {
			r.Properties = nil // GitHub serves these separately
			out = append(out, r)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("list repos of %s: no such user or organization", owner)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// GetRepo returns one repo.
func (c *Client) GetRepo(ctx context.Context, fullName string) (ghapi.Repo, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "GetRepo", fullName); err != nil {
		return ghapi.Repo{}, err
	}
	r, ok := c.Repos[fullName]
	if !ok {
		return ghapi.Repo{}, fmt.Errorf("get %s: not found", fullName)
	}
	r.Properties = nil
	return r, nil
}

// PathsExist reports which paths exist per repo.
func (c *Client) PathsExist(ctx context.Context, repos []ghapi.Repo, paths []string) (map[string][]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "PathsExist", ""); err != nil {
		return nil, err
	}
	out := map[string][]string{}
	for _, r := range repos {
		have := c.Files[r.FullName()]
		for _, p := range paths {
			if slices.Contains(have, strings.TrimPrefix(p, "/")) {
				out[r.FullName()] = append(out[r.FullName()], p)
			}
		}
	}
	return out, nil
}

// SearchCode returns the repos registered for query.
func (c *Client) SearchCode(ctx context.Context, query string) ([]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "SearchCode", ""); err != nil {
		return nil, err
	}
	for q, repos := range c.Search {
		if strings.HasPrefix(query, q) {
			return slices.Clone(repos), nil
		}
	}
	return nil, nil
}

// PRsByHead returns the newest PR from head per repo.
func (c *Client) PRsByHead(ctx context.Context, repos []string, head string) (map[string]ghapi.PR, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "PRsByHead", ""); err != nil {
		return nil, err
	}
	out := map[string]ghapi.PR{}
	for _, r := range repos {
		for i := len(c.PRs) - 1; i >= 0; i-- {
			if p := c.PRs[i]; p.Repo == r && p.HeadRef == head {
				out[r] = *p
				break
			}
		}
	}
	return out, nil
}

// SearchCampaignPRs returns PRs whose body has marker, honoring "user:"
// and "repo:" qualifiers.
func (c *Client) SearchCampaignPRs(ctx context.Context, qualifiers []string, marker string) ([]ghapi.PR, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "SearchCampaignPRs", ""); err != nil {
		return nil, err
	}
	var out []ghapi.PR
	for _, p := range c.PRs {
		if strings.Contains(p.Body, marker) && matchesQualifiers(p.Repo, qualifiers) {
			out = append(out, *p)
		}
	}
	return out, nil
}

// matchesQualifiers reports whether repo is within any user: or repo:
// qualifier; no qualifiers match everything.
func matchesQualifiers(repo string, qualifiers []string) bool {
	if len(qualifiers) == 0 {
		return true
	}
	owner, _, _ := strings.Cut(repo, "/")
	for _, q := range qualifiers {
		if v, ok := strings.CutPrefix(q, "user:"); ok && strings.EqualFold(v, owner) {
			return true
		}
		if v, ok := strings.CutPrefix(q, "repo:"); ok && strings.EqualFold(v, repo) {
			return true
		}
	}
	return false
}

// find returns the stored PR, or nil. Callers hold the lock.
func (c *Client) find(repo string, number int) *ghapi.PR {
	for _, p := range c.PRs {
		if p.Repo == repo && p.Number == number {
			return p
		}
	}
	return nil
}

// CreatePR stores a new open PR.
func (c *Client) CreatePR(ctx context.Context, repo string, in ghapi.NewPR) (ghapi.PR, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "CreatePR", repo); err != nil {
		return ghapi.PR{}, err
	}
	n := 1
	for _, p := range c.PRs {
		if p.Repo == repo {
			n++
		}
	}
	p := &ghapi.PR{
		Repo: repo, Number: n, URL: fmt.Sprintf("https://github.com/%s/pull/%d", repo, n),
		State: ghapi.StateOpen, Title: in.Title, Body: in.Body, HeadRef: in.Head, BaseRef: in.Base, Draft: in.Draft,
	}
	c.PRs = append(c.PRs, p)
	return *p, nil
}

// UpdatePR edits a stored PR.
func (c *Client) UpdatePR(ctx context.Context, repo string, number int, u ghapi.PRUpdate) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "UpdatePR", repo); err != nil {
		return err
	}
	p := c.find(repo, number)
	if p == nil {
		return fmt.Errorf("update %s#%d: not found", repo, number)
	}
	if u.Title != nil {
		p.Title = *u.Title
	}
	if u.Body != nil {
		p.Body = *u.Body
	}
	if u.State != nil {
		switch *u.State {
		case "closed":
			p.State = ghapi.StateClosed
		case "open":
			p.State = ghapi.StateOpen
		}
	}
	return nil
}

// AddLabels adds labels to a stored PR.
func (c *Client) AddLabels(ctx context.Context, repo string, number int, labels []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "AddLabels", repo); err != nil {
		return err
	}
	p := c.find(repo, number)
	if p == nil {
		return fmt.Errorf("label %s#%d: not found", repo, number)
	}
	for _, l := range labels {
		if !slices.Contains(p.Labels, l) {
			p.Labels = append(p.Labels, l)
		}
	}
	return nil
}

// RequestReviewers records requested reviewers.
func (c *Client) RequestReviewers(ctx context.Context, repo string, number int, reviewers []string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "RequestReviewers", repo); err != nil {
		return err
	}
	key := fmt.Sprintf("%s#%d", repo, number)
	c.Reviewers[key] = append(c.Reviewers[key], reviewers...)
	return nil
}

// Comment records a comment.
func (c *Client) Comment(ctx context.Context, repo string, number int, body string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "Comment", repo); err != nil {
		return err
	}
	key := fmt.Sprintf("%s#%d", repo, number)
	c.Comments[key] = append(c.Comments[key], body)
	return nil
}

// DeleteBranch records a deleted branch.
func (c *Client) DeleteBranch(ctx context.Context, repo, branch string) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "DeleteBranch", repo); err != nil {
		return err
	}
	c.DeletedBranches = append(c.DeletedBranches, repo+":"+branch)
	return nil
}

// OwnerProperties returns the Properties of every repo owned by owner.
func (c *Client) OwnerProperties(ctx context.Context, owner string) (map[string]map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "OwnerProperties", owner); err != nil {
		return nil, err
	}
	out := map[string]map[string]string{}
	for name, r := range c.Repos {
		if strings.EqualFold(r.Owner, owner) && len(r.Properties) > 0 {
			out[name] = maps.Clone(r.Properties)
		}
	}
	return out, nil
}

// RepoProperties returns one repo's Properties.
func (c *Client) RepoProperties(ctx context.Context, fullName string) (map[string]string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "RepoProperties", fullName); err != nil {
		return nil, err
	}
	return maps.Clone(c.Repos[fullName].Properties), nil
}

// RateLimits returns Limits.
func (c *Client) RateLimits(ctx context.Context) ([]ghapi.RateLimit, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if err := c.call(ctx, "RateLimits", ""); err != nil {
		return nil, err
	}
	return slices.Clone(c.Limits), nil
}

// SetPRState changes a stored PR's state, as if it were merged or closed
// on GitHub.
func (c *Client) SetPRState(repo string, number int, state, checks, review string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if p := c.find(repo, number); p != nil {
		p.State, p.Checks, p.Review = state, checks, review
	}
}

// OpenPRs returns copies of every open PR.
func (c *Client) OpenPRs() []ghapi.PR {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []ghapi.PR
	for _, p := range c.PRs {
		if p.State == ghapi.StateOpen {
			out = append(out, *p)
		}
	}
	return out
}

// CallCount returns how many times method was called.
func (c *Client) CallCount(method string) int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.Calls[method]
}

var _ ghapi.Client = (*Client)(nil)
