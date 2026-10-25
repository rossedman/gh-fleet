package ghapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/cli/go-gh/v2/pkg/api"
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

// GetRepo fetches one repository.
func (c *GitHub) GetRepo(ctx context.Context, fullName string) (Repo, error) {
	owner, name, err := SplitRepo(fullName)
	if err != nil {
		return Repo{}, err
	}
	query := `query($owner: String!, $name: String!) { repository(owner: $owner, name: $name) { ` + repoFields + ` } }`
	var resp struct {
		Repository *gqlRepo `json:"repository"`
	}
	if err := c.gql.DoWithContext(ctx, query, map[string]any{"owner": owner, "name": name}, &resp); err != nil {
		return Repo{}, fmt.Errorf("get %s: %w", fullName, err)
	}
	if resp.Repository == nil {
		return Repo{}, fmt.Errorf("get %s: not found", fullName)
	}
	return resp.Repository.toRepo(), nil
}

// gqlString quotes s as a GraphQL string literal.
func gqlString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

// PathsExist checks paths on each repo's default branch with batched
// object(expression:) lookups, BatchSize repos per query.
func (c *GitHub) PathsExist(ctx context.Context, repos []Repo, paths []string) (map[string][]string, error) {
	out := map[string][]string{}
	if len(paths) == 0 {
		return out, nil
	}
	for start := 0; start < len(repos); start += BatchSize {
		batch := repos[start:min(start+BatchSize, len(repos))]
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
		for i, r := range batch {
			objs := resp["r"+strconv.Itoa(i)]
			for j, p := range paths {
				if objs != nil && objs["p"+strconv.Itoa(j)] != nil {
					out[r.FullName()] = append(out[r.FullName()], p)
				}
			}
		}
	}
	return out, nil
}

// maxSearchPages caps search pagination; GitHub returns at most 1000 hits.
const maxSearchPages = 10

// SearchCode runs a code search and returns the unique repos with hits.
func (c *GitHub) SearchCode(ctx context.Context, query string) ([]string, error) {
	seen := map[string]bool{}
	for page := 1; page <= maxSearchPages; page++ {
		var resp struct {
			TotalCount int `json:"total_count"`
			Items      []struct {
				Repository struct {
					FullName string `json:"full_name"`
				} `json:"repository"`
			} `json:"items"`
		}
		path := "search/code?per_page=100&page=" + strconv.Itoa(page) + "&q=" + url.QueryEscape(query)
		if err := c.rest.DoWithContext(ctx, http.MethodGet, path, nil, &resp); err != nil {
			return nil, fmt.Errorf("code search: %w", err)
		}
		for _, it := range resp.Items {
			seen[it.Repository.FullName] = true
		}
		if len(resp.Items) < 100 || page*100 >= resp.TotalCount {
			break
		}
	}
	return sortedKeys(seen), nil
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
	Labels         struct {
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
  number url state title body isDraft headRefName headRefOid baseRefName reviewDecision updatedAt
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
