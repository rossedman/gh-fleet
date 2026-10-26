package ghapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/sync/errgroup"
)

// restRepo is the REST shape of a repository.
type restRepo struct {
	Name  string `json:"name"`
	Owner struct {
		Login string `json:"login"`
	} `json:"owner"`
	HTMLURL       string   `json:"html_url"`
	Archived      bool     `json:"archived"`
	Fork          bool     `json:"fork"`
	DefaultBranch string   `json:"default_branch"`
	Language      string   `json:"language"`
	Topics        []string `json:"topics"`
	Size          int      `json:"size"`
	Permissions   struct {
		Admin    bool `json:"admin"`
		Maintain bool `json:"maintain"`
		Push     bool `json:"push"`
	} `json:"permissions"`
}

// toRepo converts the REST shape to a Repo.
func (r restRepo) toRepo() Repo {
	return Repo{
		Owner:         r.Owner.Login,
		Name:          r.Name,
		URL:           r.HTMLURL,
		Archived:      r.Archived,
		Fork:          r.Fork,
		DefaultBranch: r.DefaultBranch,
		Language:      r.Language,
		Topics:        r.Topics,
		CanPush:       r.Permissions.Admin || r.Permissions.Maintain || r.Permissions.Push,
	}
}

// ListOwnerRepos lists every repository of an org or user. The REST
// listing's first page says how many pages there are, so the rest are
// fetched batchConcurrency at a time instead of one after another.
func (c *GitHub) ListOwnerRepos(ctx context.Context, owner string) ([]Repo, error) {
	items, err := restPages[restRepo](ctx, c, "orgs/"+url.PathEscape(owner)+"/repos?type=all&sort=full_name&per_page=100")
	if isNotFound(err) {
		items, err = restPages[restRepo](ctx, c, c.userReposPath(ctx, owner))
		if isNotFound(err) {
			return nil, fmt.Errorf("list repos of %s: no such user or organization", owner)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("list repos of %s: %w", owner, err)
	}
	// Pages are numbered, so a repo created mid-listing can shift one
	// onto two pages: keep each once.
	byName := map[string]Repo{}
	var maybeEmpty []string
	for _, it := range items {
		r := it.toRepo()
		byName[strings.ToLower(r.FullName())] = r
		if it.Size == 0 {
			maybeEmpty = append(maybeEmpty, r.FullName())
		}
	}
	// REST reports a default branch even for an empty repo, which has
	// none to clone. Only a repo of size 0 can be empty, so look those up.
	if len(maybeEmpty) > 0 {
		found, err := c.GetRepos(ctx, maybeEmpty)
		if err != nil {
			return nil, fmt.Errorf("list repos of %s: %w", owner, err)
		}
		for _, n := range maybeEmpty {
			k := strings.ToLower(n)
			if r, ok := found[k]; ok {
				byName[k] = r
			} else {
				delete(byName, k)
			}
		}
	}
	out := make([]Repo, 0, len(byName))
	for _, r := range byName {
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// userReposPath is the REST listing of a user's repos: the signed-in
// user's own, private ones included, or another user's public ones.
func (c *GitHub) userReposPath(ctx context.Context, owner string) string {
	var me struct {
		Login string `json:"login"`
	}
	if err := c.rest.DoWithContext(ctx, http.MethodGet, "user", nil, &me); err == nil && strings.EqualFold(me.Login, owner) {
		return "user/repos?affiliation=owner&visibility=all&sort=full_name&per_page=100"
	}
	return "users/" + url.PathEscape(owner) + "/repos?type=owner&sort=full_name&per_page=100"
}

// restPages GETs every page of a paginated REST list. The first page's
// Link header names the last page, and the others are fetched
// batchConcurrency at a time.
func restPages[T any](ctx context.Context, c *GitHub, path string) ([]T, error) {
	first, last, err := restPage[T](ctx, c, path, 1)
	if err != nil {
		return nil, err
	}
	pages := make([][]T, last+1)
	pages[1] = first
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(batchConcurrency)
	for p := 2; p <= last; p++ {
		g.Go(func() error {
			items, _, err := restPage[T](gctx, c, path, p)
			pages[p] = items
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	var out []T
	for _, p := range pages {
		out = append(out, p...)
	}
	return out, nil
}

// restPage GETs one page of a REST list, returning its items and the last
// page number.
func restPage[T any](ctx context.Context, c *GitHub, path string, page int) ([]T, int, error) {
	resp, err := c.rest.RequestWithContext(ctx, http.MethodGet, path+"&page="+strconv.Itoa(page), nil)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	var items []T
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, 0, fmt.Errorf("read %s page %d: %w", path, page, err)
	}
	return items, lastPage(resp.Header.Get("Link"), page), nil
}

// lastRel finds the last page in a Link header.
var lastRel = regexp.MustCompile(`[?&]page=(\d+)[^>]*>;\s*rel="last"`)

// lastPage returns the last page a Link header names, or page itself when
// it names none, as on the last page.
func lastPage(link string, page int) int {
	m := lastRel.FindStringSubmatch(link)
	if m == nil {
		return page
	}
	n, err := strconv.Atoi(m[1])
	if err != nil {
		return page
	}
	return max(n, page)
}
