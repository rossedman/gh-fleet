package target

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
)

// Target is one repo a campaign will touch, with why it matched.
type Target struct {
	Repo    ghapi.Repo `json:"repo"`
	Reasons []string   `json:"reasons"`
	// Vars are the repo's campaign variables, set by SetVars.
	Vars map[string]any `json:"vars,omitempty"`
}

// SetVars resolves every target's variables from a campaign's vars block.
func (r *Result) SetVars(v *campaign.Vars) {
	for i := range r.Targets {
		r.Targets[i].Vars = Vars(v, r.Targets[i].Repo)
	}
}

// Skipped is a repo that matched but cannot be changed.
type Skipped struct {
	Repo   string `json:"repo"`
	Reason string `json:"reason"`
}

// Result is the outcome of resolving targets.
type Result struct {
	Targets []Target  `json:"targets"`
	Skipped []Skipped `json:"skipped,omitempty"`
}

// Names returns the full names of the targets.
func (r *Result) Names() []string {
	out := make([]string, len(r.Targets))
	for i, t := range r.Targets {
		out[i] = t.Repo.FullName()
	}
	return out
}

// Limit keeps only the targets named in repos. It fails when a named repo
// is not a target, so a typo does not silently do nothing.
func (r *Result) Limit(repos []string) error {
	if len(repos) == 0 {
		return nil
	}
	want := map[string]bool{}
	for _, n := range repos {
		want[strings.ToLower(n)] = true
	}
	var kept []Target
	for _, t := range r.Targets {
		if want[strings.ToLower(t.Repo.FullName())] {
			kept = append(kept, t)
			delete(want, strings.ToLower(t.Repo.FullName()))
		}
	}
	if len(want) > 0 {
		missing := make([]string, 0, len(want))
		for n := range want {
			missing = append(missing, n)
		}
		sort.Strings(missing)
		return fmt.Errorf("not a campaign target: %s", strings.Join(missing, ", "))
	}
	r.Targets = kept
	return nil
}

// Resolver turns campaign targets into repos.
type Resolver struct {
	Client ghapi.Client
	// Cache stores org listings; nil disables caching.
	Cache *Cache
	// AllowReadOnly keeps repos without push access as targets, for
	// commands that only read.
	AllowReadOnly bool
	// RequireProperties makes a failure to read custom properties an
	// error. Without it, repos whose properties can't be read (for example
	// with a token that lacks access) simply have none.
	RequireProperties bool
}

// Resolve lists candidates, applies every filter and returns the targets
// sorted by full name. Repos without push access are reported as skipped
// unless AllowReadOnly is set.
func (rs *Resolver) Resolve(ctx context.Context, t campaign.Targets) (*Result, error) {
	candidates := map[string]ghapi.Repo{}
	for _, org := range t.Orgs {
		repos, err := rs.listOwner(ctx, org)
		if err != nil {
			return nil, err
		}
		for _, r := range repos {
			candidates[strings.ToLower(r.FullName())] = r
		}
	}
	listed := map[string]bool{}
	for _, o := range t.Orgs {
		listed[strings.ToLower(o)] = true
	}
	for _, name := range t.Include.Repos {
		if _, ok := candidates[strings.ToLower(name)]; ok {
			continue
		}
		if campaign.IsPattern(name) {
			// A pattern names repos of one owner: list that owner, unless
			// orgs already did, and keep the matches. Load rejects a
			// pattern in the owner without orgs to match against.
			owner, _, _ := strings.Cut(name, "/")
			if campaign.IsPattern(owner) || listed[strings.ToLower(owner)] {
				continue
			}
			listed[strings.ToLower(owner)] = true
			repos, err := rs.listOwner(ctx, owner)
			if err != nil {
				return nil, err
			}
			for _, r := range repos {
				if campaign.MatchAny(t.Include.Repos, r.FullName()) != "" {
					candidates[strings.ToLower(r.FullName())] = r
				}
			}
			continue
		}
		r, err := rs.Client.GetRepo(ctx, name)
		if err != nil {
			return nil, err
		}
		if r.Properties, err = rs.Client.RepoProperties(ctx, r.FullName()); err != nil && rs.RequireProperties {
			return nil, err
		}
		candidates[strings.ToLower(r.FullName())] = r
	}

	// Cheap metadata filters first, so file and search lookups only run
	// for repos that can still match.
	var live []ghapi.Repo
	for _, r := range candidates {
		if MetadataOnly(t, r) {
			live = append(live, r)
		}
	}
	sort.Slice(live, func(i, j int) bool { return live[i].FullName() < live[j].FullName() })

	files := map[string][]string{}
	if len(t.Include.Files) > 0 && len(live) > 0 {
		var err error
		files, err = rs.Client.PathsExist(ctx, live, t.Include.Files)
		if err != nil {
			return nil, err
		}
	}
	hits := map[string]bool{}
	if t.Include.Search != "" {
		owners := t.Orgs
		if len(owners) == 0 {
			owners = []string{""}
		}
		for _, o := range owners {
			q := t.Include.Search
			if o != "" {
				q += " org:" + o
			}
			repos, err := rs.Client.SearchCode(ctx, q)
			if err != nil {
				return nil, err
			}
			for _, n := range repos {
				hits[strings.ToLower(n)] = true
			}
		}
	}

	res := &Result{}
	for _, r := range live {
		d := Evaluate(t, r, Facts{Files: files[r.FullName()], SearchHit: hits[strings.ToLower(r.FullName())]})
		if !d.Match {
			continue
		}
		if !r.CanPush && !rs.AllowReadOnly {
			res.Skipped = append(res.Skipped, Skipped{Repo: r.FullName(), Reason: "no push access"})
			continue
		}
		res.Targets = append(res.Targets, Target{Repo: r, Reasons: d.Reasons})
	}
	return res, nil
}

// listOwner lists an owner's repos, with their custom properties, through
// the cache.
func (rs *Resolver) listOwner(ctx context.Context, owner string) ([]ghapi.Repo, error) {
	if rs.Cache != nil {
		if repos, ok := rs.Cache.Get(owner); ok {
			return repos, nil
		}
	}
	repos, err := rs.Client.ListOwnerRepos(ctx, owner)
	if err != nil {
		return nil, err
	}
	props, err := rs.Client.OwnerProperties(ctx, owner)
	if err != nil && rs.RequireProperties {
		return nil, err
	}
	for i := range repos {
		repos[i].Properties = props[repos[i].FullName()]
	}
	// A listing without the properties it should have is not cached, so a
	// later campaign that needs them fetches them again.
	if rs.Cache != nil && err == nil {
		rs.Cache.Put(owner, repos)
	}
	return repos, nil
}

// DefaultCacheTTL is how long an org listing is reused.
const DefaultCacheTTL = 10 * time.Minute

// Cache keeps org listings on disk for a short time.
type Cache struct {
	Dir string
	TTL time.Duration
	// Now returns the current time; defaults to time.Now.
	Now func() time.Time
}

// cacheVersion changes whenever cached listings gain fields, so older
// entries are refetched instead of read without them.
const cacheVersion = 2

// cacheEntry is the on-disk form of one listing.
type cacheEntry struct {
	Version int          `json:"version"`
	Fetched time.Time    `json:"fetched"`
	Repos   []ghapi.Repo `json:"repos"`
}

// now returns the cache's clock.
func (c *Cache) now() time.Time {
	if c.Now != nil {
		return c.Now()
	}
	return time.Now()
}

// path returns the cache file for owner.
func (c *Cache) path(owner string) string {
	return filepath.Join(c.Dir, strings.ToLower(filepath.Base(owner))+".json")
}

// Get returns a fresh listing for owner, if one is cached.
func (c *Cache) Get(owner string) ([]ghapi.Repo, bool) {
	b, err := os.ReadFile(c.path(owner))
	if err != nil {
		return nil, false
	}
	var e cacheEntry
	if json.Unmarshal(b, &e) != nil || e.Version != cacheVersion {
		return nil, false
	}
	ttl := c.TTL
	if ttl == 0 {
		ttl = DefaultCacheTTL
	}
	if c.now().Sub(e.Fetched) > ttl {
		return nil, false
	}
	return e.Repos, true
}

// Put stores a listing. Failures are ignored: the cache is an optimization.
func (c *Cache) Put(owner string, repos []ghapi.Repo) {
	b, err := json.Marshal(cacheEntry{Version: cacheVersion, Fetched: c.now(), Repos: repos})
	if err != nil {
		return
	}
	if os.MkdirAll(c.Dir, 0o700) != nil {
		return
	}
	tmp := c.path(owner) + ".tmp"
	if os.WriteFile(tmp, b, 0o600) == nil {
		_ = os.Rename(tmp, c.path(owner))
	}
}
