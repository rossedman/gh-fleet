// Package ghapi is the thin layer between gh fleet and the GitHub API.
//
// Everything above this package talks to GitHub through the Client
// interface, so it can be tested with the in-memory fake in ghapi/fake.
// The real implementation uses the REST and GraphQL clients from go-gh,
// behind a Transport that enforces write rate limits and pauses every
// worker when GitHub reports a secondary rate limit.
package ghapi

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// Repo is a repository as seen by target resolution.
type Repo struct {
	Owner         string   `json:"owner"`
	Name          string   `json:"name"`
	DefaultBranch string   `json:"defaultBranch"`
	Topics        []string `json:"topics,omitempty"`
	Language      string   `json:"language,omitempty"`
	Archived      bool     `json:"archived"`
	Fork          bool     `json:"fork"`
	CanPush       bool     `json:"canPush"`
	URL           string   `json:"url,omitempty"`
	// Properties are the repository's custom property values, set by its
	// organization (multi-select values are comma-joined).
	Properties map[string]string `json:"properties,omitempty"`
}

// FullName returns owner/name.
func (r Repo) FullName() string { return r.Owner + "/" + r.Name }

// PR states as reported by GitHub's GraphQL API.
const (
	StateOpen   = "OPEN"
	StateClosed = "CLOSED"
	StateMerged = "MERGED"
)

// PR is a pull request opened by a campaign.
type PR struct {
	Repo      string    `json:"repo"`
	Number    int       `json:"number"`
	URL       string    `json:"url"`
	State     string    `json:"state"`
	Title     string    `json:"title"`
	Body      string    `json:"-"`
	HeadRef   string    `json:"headRef"`
	HeadSHA   string    `json:"headSha,omitempty"`
	BaseRef   string    `json:"baseRef,omitempty"`
	Draft     bool      `json:"draft"`
	Labels    []string  `json:"labels,omitempty"`
	Checks    string    `json:"checks,omitempty"`
	Review    string    `json:"review,omitempty"`
	UpdatedAt time.Time `json:"updatedAt,omitzero"`
	// MergeState is GitHub's mergeStateStatus: CLEAN, BEHIND, BLOCKED,
	// DIRTY, DRAFT, HAS_HOOKS, UNSTABLE or UNKNOWN.
	MergeState string `json:"mergeState,omitempty"`
	// AutoMerge is set when auto-merge is enabled.
	AutoMerge bool `json:"autoMerge,omitempty"`
	// NodeID is the GraphQL ID, needed to enable auto-merge.
	NodeID string `json:"-"`
}

// Merge methods, as the REST API names them.
const (
	MergeMethodMerge  = "merge"
	MergeMethodSquash = "squash"
	MergeMethodRebase = "rebase"
)

// NewPR is the input for creating a pull request.
type NewPR struct {
	Title string
	Body  string
	Head  string
	Base  string
	Draft bool
}

// PRUpdate lists the fields to change on a pull request; nil means keep.
type PRUpdate struct {
	Title *string
	Body  *string
	State *string // "open" or "closed"
}

// RateLimit is the remaining primary quota for one API resource.
type RateLimit struct {
	Resource  string    `json:"resource"`
	Limit     int       `json:"limit"`
	Remaining int       `json:"remaining"`
	Reset     time.Time `json:"reset"`
}

// Client is every GitHub operation gh fleet performs.
type Client interface {
	// ListOwnerRepos lists every repository of an organization or user.
	ListOwnerRepos(ctx context.Context, owner string) ([]Repo, error)
	// GetRepo fetches one repository by owner/name.
	GetRepo(ctx context.Context, fullName string) (Repo, error)
	// PathsExist reports, per repo full name, which of paths exist on the
	// repo's default branch.
	PathsExist(ctx context.Context, repos []Repo, paths []string) (map[string][]string, error)
	// SearchCode returns the full names of repos with a code search hit.
	SearchCode(ctx context.Context, query string) ([]string, error)
	// PRsByHead returns, per repo full name, the most recent PR from head
	// in any state, with checks and review state. Repos without one are
	// absent from the map. Queries are batched, BatchSize repos each.
	PRsByHead(ctx context.Context, repos []string, head string) (map[string]PR, error)
	// SearchCampaignPRs finds PRs whose body contains marker, using search
	// qualifiers such as "user:acme" or "repo:acme/api" to scope it.
	SearchCampaignPRs(ctx context.Context, qualifiers []string, marker string) ([]PR, error)
	// CreatePR opens a pull request.
	CreatePR(ctx context.Context, repo string, in NewPR) (PR, error)
	// UpdatePR changes a pull request's title, body or state.
	UpdatePR(ctx context.Context, repo string, number int, u PRUpdate) error
	// AddLabels adds labels to a pull request.
	AddLabels(ctx context.Context, repo string, number int, labels []string) error
	// RequestReviewers requests review from users and org/team slugs.
	RequestReviewers(ctx context.Context, repo string, number int, reviewers []string) error
	// Comment posts a comment on a pull request.
	Comment(ctx context.Context, repo string, number int, body string) error
	// DeleteBranch deletes a branch.
	DeleteBranch(ctx context.Context, repo, branch string) error
	// MergePR merges a pull request with method, only if its head is
	// still sha.
	MergePR(ctx context.Context, repo string, number int, method, sha string) error
	// EnableAutoMerge turns on auto-merge, so GitHub merges the PR once
	// its requirements pass.
	EnableAutoMerge(ctx context.Context, pr PR, method string) error
	// UpdateBranch merges the base branch into a PR's head, only if its
	// head is still sha.
	UpdateBranch(ctx context.Context, repo string, number int, sha string) error
	// RateLimits returns the remaining primary quota per resource.
	RateLimits(ctx context.Context) ([]RateLimit, error)
	// OwnerProperties returns custom property values for every repo of an
	// organization, keyed by repo full name. Owners without custom
	// properties (users, or orgs that define none) return an empty map.
	OwnerProperties(ctx context.Context, owner string) (map[string]map[string]string, error)
	// RepoProperties returns one repo's custom property values.
	RepoProperties(ctx context.Context, fullName string) (map[string]string, error)
}

// SplitRepo splits owner/name, failing on anything else.
func SplitRepo(full string) (owner, name string, err error) {
	owner, name, ok := strings.Cut(full, "/")
	if !ok || owner == "" || name == "" || strings.Contains(name, "/") {
		return "", "", fmt.Errorf("invalid repository %q, want owner/name", full)
	}
	return owner, name, nil
}

// SplitReviewers separates users from org/team slugs, returning users and
// team slugs (the part after the slash).
func SplitReviewers(reviewers []string) (users, teams []string) {
	for _, r := range reviewers {
		if _, team, ok := strings.Cut(r, "/"); ok {
			teams = append(teams, team)
		} else {
			users = append(users, r)
		}
	}
	return users, teams
}
