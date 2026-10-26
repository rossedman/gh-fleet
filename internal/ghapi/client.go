package ghapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
	"golang.org/x/sync/errgroup"
)

// BatchSize is how many repos or PRs go into one batched GraphQL query.
const BatchSize = 50

// GitHub is the Client backed by the real GitHub API through go-gh.
type GitHub struct {
	rest *api.RESTClient
	gql  *api.GraphQLClient
	// Transport is the shared rate limiting transport.
	Transport *Transport
}

// Options configures New.
type Options struct {
	// Host is the GitHub host, e.g. github.com. Empty uses gh's default.
	Host string
	// AuthToken overrides the token gh would use.
	AuthToken string
	// WriteRate is content-creating requests per second (default 1).
	WriteRate float64
	// Base is the transport requests finally go through (for tests).
	Base http.RoundTripper
}

// ErrAuthRequired is returned when gh has no token for the host. Its
// message is gh's own, so the fix reads the same.
var ErrAuthRequired = errors.New("To get started with GitHub CLI, please run:  gh auth login\n" + //nolint:staticcheck // gh's wording
	"Alternatively, populate the GH_TOKEN environment variable with a GitHub API authentication token.")

// New builds a GitHub client that authenticates the way gh does.
func New(opts Options) (*GitHub, error) {
	rate := opts.WriteRate
	if rate == 0 {
		rate = DefaultWriteRate
	}
	tr := NewTransport(opts.Base, rate)
	co := api.ClientOptions{Host: opts.Host, AuthToken: opts.AuthToken, Transport: tr}
	rest, err := api.NewRESTClient(co)
	if err != nil {
		if strings.HasPrefix(err.Error(), "authentication token not found") {
			return nil, ErrAuthRequired
		}
		return nil, err
	}
	gql, err := api.NewGraphQLClient(co)
	if err != nil {
		return nil, err
	}
	return &GitHub{rest: rest, gql: gql, Transport: tr}, nil
}

// repoFields is the GraphQL selection for a Repo.
const repoFields = `name owner { login } url isArchived isFork viewerPermission
defaultBranchRef { name } primaryLanguage { name }
repositoryTopics(first: 50) { nodes { topic { name } } }`

// gqlRepo is the GraphQL shape of repoFields.
type gqlRepo struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	URL              string `json:"url"`
	IsArchived       bool   `json:"isArchived"`
	IsFork           bool   `json:"isFork"`
	ViewerPermission string `json:"viewerPermission"`
	DefaultBranchRef *struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
	PrimaryLanguage *struct {
		Name string `json:"name"`
	} `json:"primaryLanguage"`
	RepositoryTopics struct {
		Nodes []struct {
			Topic struct {
				Name string `json:"name"`
			} `json:"topic"`
		} `json:"nodes"`
	} `json:"repositoryTopics"`
}

// toRepo converts the GraphQL shape to a Repo.
func (g gqlRepo) toRepo() Repo {
	r := Repo{
		Owner:    g.Owner.Login,
		Name:     g.Name,
		URL:      g.URL,
		Archived: g.IsArchived,
		Fork:     g.IsFork,
	}
	switch g.ViewerPermission {
	case "ADMIN", "MAINTAIN", "WRITE":
		r.CanPush = true
	}
	if g.DefaultBranchRef != nil {
		r.DefaultBranch = g.DefaultBranchRef.Name
	}
	if g.PrimaryLanguage != nil {
		r.Language = g.PrimaryLanguage.Name
	}
	for _, n := range g.RepositoryTopics.Nodes {
		r.Topics = append(r.Topics, n.Topic.Name)
	}
	return r
}

// ListOwnerRepos pages through every repository of an org or user.
func (c *GitHub) ListOwnerRepos(ctx context.Context, owner string) ([]Repo, error) {
	query := `query($login: String!, $after: String) {
  repositoryOwner(login: $login) {
    repositories(first: 100, after: $after, orderBy: {field: NAME, direction: ASC}) {
      pageInfo { hasNextPage endCursor }
      nodes { ` + repoFields + ` }
    }
  }
}`
	var out []Repo
	var after *string
	for {
		var resp struct {
			RepositoryOwner *struct {
				Repositories struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []gqlRepo `json:"nodes"`
				} `json:"repositories"`
			} `json:"repositoryOwner"`
		}
		vars := map[string]any{"login": owner, "after": after}
		if err := c.gql.DoWithContext(ctx, query, vars, &resp); err != nil {
			return nil, fmt.Errorf("list repos of %s: %w", owner, err)
		}
		if resp.RepositoryOwner == nil {
			return nil, fmt.Errorf("list repos of %s: no such user or organization", owner)
		}
		for _, n := range resp.RepositoryOwner.Repositories.Nodes {
			out = append(out, n.toRepo())
		}
		pi := resp.RepositoryOwner.Repositories.PageInfo
		if !pi.HasNextPage {
			return out, nil
		}
		cursor := pi.EndCursor
		after = &cursor
	}
}

// ErrNotFound is returned when a repository doesn't exist, or the token
// can't see it.
var ErrNotFound = errors.New("not found")

// GetRepos looks up repositories by owner/name, BatchSize per GraphQL
// query. The result is keyed by the lower-cased name asked for; a repo that
// doesn't exist, or that the token can't see, is absent.
func (c *GitHub) GetRepos(ctx context.Context, names []string) (map[string]Repo, error) {
	out := map[string]Repo{}
	for start := 0; start < len(names); start += BatchSize {
		batch := names[start:min(start+BatchSize, len(names))]
		var q strings.Builder
		q.WriteString("query {")
		for i, n := range batch {
			owner, name, err := SplitRepo(n)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&q, " r%d: repository(owner: %s, name: %s) { %s }", i, gqlString(owner), gqlString(name), repoFields)
		}
		q.WriteString(" }")
		var resp map[string]*gqlRepo
		if err := c.gql.DoWithContext(ctx, q.String(), nil, &resp); err != nil && !onlyNotFound(err) {
			return nil, fmt.Errorf("look up repos: %w", err)
		}
		for i, n := range batch {
			if r := resp["r"+strconv.Itoa(i)]; r != nil {
				out[strings.ToLower(n)] = r.toRepo()
			}
		}
	}
	return out, nil
}

// onlyNotFound reports whether err is a GraphQL error made only of
// NOT_FOUND items, which GitHub sends beside the data of the repos it
// found.
func onlyNotFound(err error) bool {
	var gqlErr *api.GraphQLError
	if !errors.As(err, &gqlErr) {
		return false
	}
	for _, e := range gqlErr.Errors {
		if e.Type != "NOT_FOUND" {
			return false
		}
	}
	return true
}

// gqlString quotes s as a GraphQL string literal.
func gqlString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// batchConcurrency is how many batched read queries run at once.
const batchConcurrency = 4

// PathsExist checks paths on each repo's default branch with batched
// object(expression:) lookups, BatchSize repos per query and
// batchConcurrency queries at a time.
func (c *GitHub) PathsExist(ctx context.Context, repos []Repo, paths []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(paths) == 0 {
		return out, nil
	}
	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(batchConcurrency)
	for start := 0; start < len(repos); start += BatchSize {
		batch := repos[start:min(start+BatchSize, len(repos))]
		g.Go(func() error {
			found, err := c.pathsExist(gctx, batch, paths)
			if err != nil {
				return err
			}
			mu.Lock()
			maps.Copy(out, found)
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	return out, nil
}

// pathsExist checks paths on one batch of repos in a single query.
func (c *GitHub) pathsExist(ctx context.Context, batch []Repo, paths []string) (map[string][]string, error) {
	var q strings.Builder
	q.WriteString("query {")
	for i, r := range batch {
		ref := r.DefaultBranch
		if ref == "" {
			ref = "HEAD"
		}
		fmt.Fprintf(&q, " r%d: repository(owner: %s, name: %s) {", i, gqlString(r.Owner), gqlString(r.Name))
		for j, p := range paths {
			fmt.Fprintf(&q, " p%d: object(expression: %s) { __typename }", j, gqlString(ref+":"+strings.TrimPrefix(p, "/")))
		}
		q.WriteString(" }")
	}
	q.WriteString(" }")
	var resp map[string]map[string]*struct {
		Typename string `json:"__typename"`
	}
	if err := c.gql.DoWithContext(ctx, q.String(), nil, &resp); err != nil {
		return nil, fmt.Errorf("check paths: %w", err)
	}
	out := map[string][]string{}
	for i, r := range batch {
		objs := resp["r"+strconv.Itoa(i)]
		for j, p := range paths {
			if objs != nil && objs["p"+strconv.Itoa(j)] != nil {
				out[r.FullName()] = append(out[r.FullName()], p)
			}
		}
	}
	return out, nil
}

// maxSearchPages caps search pagination; GitHub returns at most 1000 hits.
const maxSearchPages = 10

// codeSearchCap is the most results GitHub returns for one search query.
const codeSearchCap = maxSearchPages * 100

// maxIndexedFileSize is the largest file code search indexes, in bytes.
const maxIndexedFileSize = 384 * 1024

// SearchCode runs a code search and returns the unique repos with hits.
// GitHub returns at most codeSearchCap files per query, counting every
// matching file rather than every repo, so a query with more matches is
// split by file size until each part fits.
func (c *GitHub) SearchCode(ctx context.Context, query string) (CodeSearch, error) {
	seen := map[string]bool{}
	partial, err := c.searchCode(ctx, query, 0, maxIndexedFileSize, false, seen)
	if err != nil {
		return CodeSearch{}, err
	}
	return CodeSearch{Repos: sortedKeys(seen), Partial: partial}, nil
}

// searchCode adds the repos with hits for query to seen, among files of
// lo to hi bytes when sized. It reports whether some hits could not be
// returned.
func (c *GitHub) searchCode(ctx context.Context, query string, lo, hi int, sized bool, seen map[string]bool) (bool, error) {
	q := query
	if sized {
		q += fmt.Sprintf(" size:%d..%d", lo, hi)
	}
	partial := false
	total := 0
	for page := 1; page <= maxSearchPages; page++ {
		var resp struct {
			TotalCount        int  `json:"total_count"`
			IncompleteResults bool `json:"incomplete_results"`
			Items             []struct {
				Repository struct {
					FullName string `json:"full_name"`
				} `json:"repository"`
			} `json:"items"`
		}
		path := "search/code?per_page=100&page=" + strconv.Itoa(page) + "&q=" + url.QueryEscape(q)
		if err := c.rest.DoWithContext(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return false, fmt.Errorf("code search: %w", err)
		}
		for _, it := range resp.Items {
			seen[it.Repository.FullName] = true
		}
		partial = partial || resp.IncompleteResults
		total = resp.TotalCount
		// Too many matches for one query: search each half of the size
		// range instead. A query that sets its own size can't be split.
		if page == 1 && total > codeSearchCap && lo < hi && !strings.Contains(query, "size:") {
			mid := splitSize(lo, hi)
			p1, err := c.searchCode(ctx, query, lo, mid, true, seen)
			if err != nil {
				return false, err
			}
			p2, err := c.searchCode(ctx, query, mid+1, hi, true, seen)
			if err != nil {
				return false, err
			}
			return partial || p1 || p2, nil
		}
		if len(resp.Items) < 100 || page*100 >= total {
			break
		}
	}
	return partial || total > codeSearchCap, nil
}

// splitSize picks where to split a file size range. Most files are small,
// so the split is geometric: 0..393216 splits at 626, then 0..626 at 24.
func splitSize(lo, hi int) int {
	mid := int(math.Sqrt(float64(lo+1)*float64(hi+1))) - 1
	return min(max(mid, lo), hi-1)
}

// sortedKeys returns the keys of a set in order.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// restPR is the REST shape of a pull request.
type restPR struct {
	Number   int     `json:"number"`
	HTMLURL  string  `json:"html_url"`
	State    string  `json:"state"`
	Title    string  `json:"title"`
	Body     string  `json:"body"`
	Draft    bool    `json:"draft"`
	MergedAt *string `json:"merged_at"`
	Head     struct {
		Ref string `json:"ref"`
		SHA string `json:"sha"`
	} `json:"head"`
	Base struct {
		Ref string `json:"ref"`
	} `json:"base"`
	Labels []struct {
		Name string `json:"name"`
	} `json:"labels"`
	UpdatedAt time.Time `json:"updated_at"`
}

// toPR converts the REST shape to a PR.
func (p restPR) toPR(repo string) PR {
	out := PR{
		Repo: repo, Number: p.Number, URL: p.HTMLURL, Title: p.Title, Body: p.Body,
		HeadRef: p.Head.Ref, HeadSHA: p.Head.SHA, BaseRef: p.Base.Ref, Draft: p.Draft, UpdatedAt: p.UpdatedAt,
	}
	switch {
	case p.MergedAt != nil:
		out.State = StateMerged
	case p.State == "closed":
		out.State = StateClosed
	default:
		out.State = StateOpen
	}
	for _, l := range p.Labels {
		out.Labels = append(out.Labels, l.Name)
	}
	return out
}

// SearchCampaignPRs searches open and closed PRs whose body mentions the
// marker and keeps exact matches. Open and closed are searched separately
// so each gets the full 1000-result window.
func (c *GitHub) SearchCampaignPRs(ctx context.Context, qualifiers []string, marker string) ([]PR, error) {
	term := strings.TrimSuffix(strings.TrimPrefix(marker, "<!-- "), " -->")
	var out []PR
	seen := map[string]bool{}
	for _, state := range []string{"is:open", "is:closed"} {
		q := strings.Join(append([]string{"is:pr", state, "in:body", gqlString(term)}, qualifiers...), " ")
		for page := 1; page <= maxSearchPages; page++ {
			var resp struct {
				TotalCount int `json:"total_count"`
				Items      []struct {
					Number        int    `json:"number"`
					HTMLURL       string `json:"html_url"`
					State         string `json:"state"`
					Title         string `json:"title"`
					Body          string `json:"body"`
					Draft         bool   `json:"draft"`
					RepositoryURL string `json:"repository_url"`
					PullRequest   struct {
						MergedAt *string `json:"merged_at"`
					} `json:"pull_request"`
					UpdatedAt time.Time `json:"updated_at"`
				} `json:"items"`
			}
			path := "search/issues?per_page=100&page=" + strconv.Itoa(page) + "&q=" + url.QueryEscape(q)
			if err := c.rest.DoWithContext(ctx, http.MethodGet, path, nil, &resp); err != nil {
				return nil, fmt.Errorf("search PRs: %w", err)
			}
			for _, it := range resp.Items {
				if !strings.Contains(it.Body, marker) {
					continue
				}
				repo := repoFromAPIURL(it.RepositoryURL)
				key := repo + "#" + strconv.Itoa(it.Number)
				if seen[key] {
					continue
				}
				seen[key] = true
				pr := PR{Repo: repo, Number: it.Number, URL: it.HTMLURL, Title: it.Title, Body: it.Body, Draft: it.Draft, UpdatedAt: it.UpdatedAt, State: StateOpen}
				switch {
				case it.PullRequest.MergedAt != nil:
					pr.State = StateMerged
				case it.State == "closed":
					pr.State = StateClosed
				}
				out = append(out, pr)
			}
			if len(resp.Items) < 100 || page*100 >= resp.TotalCount {
				break
			}
		}
	}
	return out, nil
}

// repoFromAPIURL turns https://api.github.com/repos/o/r into o/r.
func repoFromAPIURL(u string) string {
	if _, rest, ok := strings.Cut(u, "/repos/"); ok {
		return rest
	}
	return u
}

// gqlPR is the GraphQL shape of a pull request with its checks.
type gqlPR struct {
	Number         int       `json:"number"`
	URL            string    `json:"url"`
	State          string    `json:"state"`
	Title          string    `json:"title"`
	Body           string    `json:"body"`
	IsDraft        bool      `json:"isDraft"`
	HeadRefName    string    `json:"headRefName"`
	HeadRefOid     string    `json:"headRefOid"`
	BaseRefName    string    `json:"baseRefName"`
	ReviewDecision string    `json:"reviewDecision"`
	UpdatedAt      time.Time `json:"updatedAt"`
	ID             string    `json:"id"`
	MergeState     string    `json:"mergeStateStatus"`
	AutoMerge      *struct {
		EnabledAt time.Time `json:"enabledAt"`
	} `json:"autoMergeRequest"`
	Labels struct {
		Nodes []struct {
			Name string `json:"name"`
		} `json:"nodes"`
	} `json:"labels"`
	Commits struct {
		Nodes []struct {
			Commit struct {
				StatusCheckRollup *struct {
					State string `json:"state"`
				} `json:"statusCheckRollup"`
			} `json:"commit"`
		} `json:"nodes"`
	} `json:"commits"`
}

// toPR converts the GraphQL shape to a PR.
func (p gqlPR) toPR(repo string) PR {
	pr := PR{
		Repo: repo, Number: p.Number, URL: p.URL, State: p.State, Title: p.Title, Body: p.Body, Draft: p.IsDraft,
		HeadRef: p.HeadRefName, HeadSHA: p.HeadRefOid, BaseRef: p.BaseRefName, Review: p.ReviewDecision, UpdatedAt: p.UpdatedAt,
		NodeID: p.ID, MergeState: p.MergeState, AutoMerge: p.AutoMerge != nil,
	}
	for _, l := range p.Labels.Nodes {
		pr.Labels = append(pr.Labels, l.Name)
	}
	if len(p.Commits.Nodes) > 0 && p.Commits.Nodes[0].Commit.StatusCheckRollup != nil {
		pr.Checks = p.Commits.Nodes[0].Commit.StatusCheckRollup.State
	}
	return pr
}

// PRsByHead looks up the newest PR from head in each repo, BatchSize repos
// per GraphQL query.
func (c *GitHub) PRsByHead(ctx context.Context, repos []string, head string) (map[string]PR, error) {
	out := map[string]PR{}
	for start := 0; start < len(repos); start += BatchSize {
		batch := repos[start:min(start+BatchSize, len(repos))]
		var q strings.Builder
		q.WriteString("query {")
		for i, r := range batch {
			owner, name, err := SplitRepo(r)
			if err != nil {
				return nil, err
			}
			fmt.Fprintf(&q, ` r%d: repository(owner: %s, name: %s) { pullRequests(headRefName: %s, first: 1, orderBy: {field: CREATED_AT, direction: DESC}) { nodes {
  id number url state title body isDraft headRefName headRefOid baseRefName reviewDecision updatedAt mergeStateStatus
  autoMergeRequest { enabledAt }
  labels(first: 50) { nodes { name } }
  commits(last: 1) { nodes { commit { statusCheckRollup { state } } } }
} } }`, i, gqlString(owner), gqlString(name), gqlString(head))
		}
		q.WriteString(" }")
		var resp map[string]*struct {
			PullRequests struct {
				Nodes []gqlPR `json:"nodes"`
			} `json:"pullRequests"`
		}
		if err := c.gql.DoWithContext(ctx, q.String(), nil, &resp); err != nil {
			return nil, fmt.Errorf("look up PRs: %w", err)
		}
		for i, r := range batch {
			node := resp["r"+strconv.Itoa(i)]
			if node == nil || len(node.PullRequests.Nodes) == 0 {
				continue
			}
			out[r] = node.PullRequests.Nodes[0].toPR(r)
		}
	}
	return out, nil
}

// jsonBody encodes v as a request body.
func jsonBody(v any) *bytes.Reader {
	b, _ := json.Marshal(v)
	return bytes.NewReader(b)
}

// CreatePR opens a pull request.
func (c *GitHub) CreatePR(ctx context.Context, repo string, in NewPR) (PR, error) {
	var resp restPR
	body := map[string]any{"title": in.Title, "body": in.Body, "head": in.Head, "base": in.Base, "draft": in.Draft}
	if err := c.rest.DoWithContext(ctx, http.MethodPost, "repos/"+repo+"/pulls", jsonBody(body), &resp); err != nil {
		return PR{}, fmt.Errorf("create PR in %s: %w", repo, err)
	}
	return resp.toPR(repo), nil
}

// UpdatePR edits a pull request.
func (c *GitHub) UpdatePR(ctx context.Context, repo string, number int, u PRUpdate) error {
	body := map[string]any{}
	if u.Title != nil {
		body["title"] = *u.Title
	}
	if u.Body != nil {
		body["body"] = *u.Body
	}
	if u.State != nil {
		body["state"] = *u.State
	}
	if len(body) == 0 {
		return nil
	}
	path := fmt.Sprintf("repos/%s/pulls/%d", repo, number)
	if err := c.rest.DoWithContext(ctx, http.MethodPatch, path, jsonBody(body), nil); err != nil {
		return fmt.Errorf("update %s#%d: %w", repo, number, err)
	}
	return nil
}

// AddLabels adds labels to a PR.
func (c *GitHub) AddLabels(ctx context.Context, repo string, number int, labels []string) error {
	if len(labels) == 0 {
		return nil
	}
	path := fmt.Sprintf("repos/%s/issues/%d/labels", repo, number)
	if err := c.rest.DoWithContext(ctx, http.MethodPost, path, jsonBody(map[string]any{"labels": labels}), nil); err != nil {
		return fmt.Errorf("label %s#%d: %w", repo, number, err)
	}
	return nil
}

// RequestReviewers requests reviews from users and teams.
func (c *GitHub) RequestReviewers(ctx context.Context, repo string, number int, reviewers []string) error {
	if len(reviewers) == 0 {
		return nil
	}
	users, teams := SplitReviewers(reviewers)
	body := map[string]any{"reviewers": nonNil(users), "team_reviewers": nonNil(teams)}
	path := fmt.Sprintf("repos/%s/pulls/%d/requested_reviewers", repo, number)
	if err := c.rest.DoWithContext(ctx, http.MethodPost, path, jsonBody(body), nil); err != nil {
		return fmt.Errorf("request reviewers on %s#%d: %w", repo, number, err)
	}
	return nil
}

// nonNil returns s, or an empty slice so it encodes as [] instead of null.
func nonNil(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

// Comment posts a comment on a PR.
func (c *GitHub) Comment(ctx context.Context, repo string, number int, body string) error {
	path := fmt.Sprintf("repos/%s/issues/%d/comments", repo, number)
	if err := c.rest.DoWithContext(ctx, http.MethodPost, path, jsonBody(map[string]any{"body": body}), nil); err != nil {
		return fmt.Errorf("comment on %s#%d: %w", repo, number, err)
	}
	return nil
}

// DeleteBranch deletes a branch; a branch that is already gone is not an
// error.
func (c *GitHub) DeleteBranch(ctx context.Context, repo, branch string) error {
	path := fmt.Sprintf("repos/%s/git/refs/heads/%s", repo, branch)
	err := c.rest.DoWithContext(ctx, http.MethodDelete, path, nil, nil)
	var he *api.HTTPError
	if err != nil && errors.As(err, &he) && (he.StatusCode == http.StatusNotFound || he.StatusCode == http.StatusUnprocessableEntity) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("delete branch %s in %s: %w", branch, repo, err)
	}
	return nil
}

// MergePR merges a pull request. GitHub refuses when the head moved past
// sha or the PR can't be merged.
func (c *GitHub) MergePR(ctx context.Context, repo string, number int, method, sha string) error {
	path := fmt.Sprintf("repos/%s/pulls/%d/merge", repo, number)
	body := map[string]any{"merge_method": method}
	if sha != "" {
		body["sha"] = sha
	}
	if err := c.rest.DoWithContext(ctx, http.MethodPut, path, jsonBody(body), nil); err != nil {
		return fmt.Errorf("merge %s#%d: %w", repo, number, err)
	}
	return nil
}

// EnableAutoMerge turns on auto-merge for a pull request.
func (c *GitHub) EnableAutoMerge(ctx context.Context, pr PR, method string) error {
	if pr.NodeID == "" {
		return fmt.Errorf("enable auto-merge on %s#%d: no node ID", pr.Repo, pr.Number)
	}
	const mutation = `mutation($id: ID!, $method: PullRequestMergeMethod!) {
  enablePullRequestAutoMerge(input: {pullRequestId: $id, mergeMethod: $method}) { clientMutationId }
}`
	vars := map[string]any{"id": pr.NodeID, "method": strings.ToUpper(method)}
	var resp struct{}
	if err := c.gql.DoWithContext(ctx, mutation, vars, &resp); err != nil {
		return fmt.Errorf("enable auto-merge on %s#%d: %w", pr.Repo, pr.Number, err)
	}
	return nil
}

// UpdateBranch merges the base branch into a pull request's head.
func (c *GitHub) UpdateBranch(ctx context.Context, repo string, number int, sha string) error {
	path := fmt.Sprintf("repos/%s/pulls/%d/update-branch", repo, number)
	body := map[string]any{}
	if sha != "" {
		body["expected_head_sha"] = sha
	}
	if err := c.rest.DoWithContext(ctx, http.MethodPut, path, jsonBody(body), nil); err != nil {
		return fmt.Errorf("update branch of %s#%d: %w", repo, number, err)
	}
	return nil
}

// restProperty is one custom property value as REST returns it.
type restProperty struct {
	Name  string `json:"property_name"`
	Value any    `json:"value"`
}

// propertyMap flattens REST property values into strings. Multi-select
// values are comma-joined and unset values are left out.
func propertyMap(props []restProperty) map[string]string {
	out := map[string]string{}
	for _, p := range props {
		switch v := p.Value.(type) {
		case nil:
		case string:
			out[p.Name] = v
		case []any:
			parts := make([]string, 0, len(v))
			for _, e := range v {
				parts = append(parts, fmt.Sprint(e))
			}
			out[p.Name] = strings.Join(parts, ",")
		default:
			out[p.Name] = fmt.Sprint(v)
		}
	}
	return out
}

// isNotFound reports whether err is an HTTP 404 from the API.
func isNotFound(err error) bool {
	var he *api.HTTPError
	return errors.As(err, &he) && he.StatusCode == http.StatusNotFound
}

// OwnerProperties pages through an organization's custom property values.
// A 404 (a user account, or no access to properties) means none.
func (c *GitHub) OwnerProperties(ctx context.Context, owner string) (map[string]map[string]string, error) {
	out := map[string]map[string]string{}
	for page := 1; ; page++ {
		var resp []struct {
			FullName   string         `json:"repository_full_name"`
			Properties []restProperty `json:"properties"`
		}
		path := fmt.Sprintf("orgs/%s/properties/values?per_page=100&page=%d", url.PathEscape(owner), page)
		if err := c.rest.DoWithContext(ctx, http.MethodGet, path, nil, &resp); err != nil {
			if isNotFound(err) {
				return out, nil
			}
			return nil, fmt.Errorf("custom properties of %s: %w", owner, err)
		}
		for _, r := range resp {
			if m := propertyMap(r.Properties); len(m) > 0 {
				out[r.FullName] = m
			}
		}
		if len(resp) < 100 {
			return out, nil
		}
	}
}

// RepoProperties returns one repo's custom property values.
func (c *GitHub) RepoProperties(ctx context.Context, fullName string) (map[string]string, error) {
	if _, _, err := SplitRepo(fullName); err != nil {
		return nil, err
	}
	var resp []restProperty
	if err := c.rest.DoWithContext(ctx, http.MethodGet, "repos/"+fullName+"/properties/values", nil, &resp); err != nil {
		if isNotFound(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("custom properties of %s: %w", fullName, err)
	}
	return propertyMap(resp), nil
}

// RateLimits reports the core, GraphQL and search quotas.
func (c *GitHub) RateLimits(ctx context.Context) ([]RateLimit, error) {
	var resp struct {
		Resources map[string]struct {
			Limit     int   `json:"limit"`
			Remaining int   `json:"remaining"`
			Reset     int64 `json:"reset"`
		} `json:"resources"`
	}
	if err := c.rest.DoWithContext(ctx, http.MethodGet, "rate_limit", nil, &resp); err != nil {
		return nil, fmt.Errorf("rate limit: %w", err)
	}
	var out []RateLimit
	for _, name := range []string{"core", "graphql", "search", "code_search"} {
		r, ok := resp.Resources[name]
		if !ok {
			continue
		}
		out = append(out, RateLimit{Resource: name, Limit: r.Limit, Remaining: r.Remaining, Reset: time.Unix(r.Reset, 0)})
	}
	return out, nil
}
