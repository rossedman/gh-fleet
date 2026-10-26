package target

import (
	"context"
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"golang.org/x/sync/errgroup"
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
	// Only, when set, limits resolution to these owner/name repos: they
	// are looked up one by one instead of listing every org.
	Only []string
	// Progress, when set, is told what resolution is doing, so a long
	// listing or file check isn't silent.
	Progress func(msg string)
	// RequireProperties fetches custom properties, for campaigns that use
	// them, and makes a failure to read them an error. Without it, repos
	// have none, which saves a call per 100 repos of each org.
	RequireProperties bool

	// props is set while resolving when properties are fetched: the
	// campaign requires them, or its targets filter on them.
	props bool
}

// Resolve lists candidates, applies every filter and returns the targets
// sorted by full name. Repos without push access are reported as skipped
// unless AllowReadOnly is set.
func (rs *Resolver) Resolve(ctx context.Context, t campaign.Targets) (*Result, error) {
	rs.props = rs.RequireProperties || len(t.Include.Properties) > 0 || len(t.Exclude.Properties) > 0
	var candidates map[string]ghapi.Repo
	var err error
	if len(rs.Only) > 0 {
		candidates, err = rs.named(ctx, t)
	} else {
		candidates, err = rs.candidates(ctx, t)
	}
	if err != nil {
		return nil, err
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
		rs.progress(fmt.Sprintf("Checking %s in %s", strings.Join(t.Include.Files, ", "), countRepos(len(live))))
		var err error
		files, err = rs.Client.PathsExist(ctx, live, t.Include.Files)
		if err != nil {
			return nil, err
		}
	}
	hits := map[string]bool{}
	if t.Include.Search != "" {
		rs.progress(fmt.Sprintf("Searching code for %q", t.Include.Search))
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
		if r.DefaultBranch == "" {
			// GitHub has no default branch for an empty repo, so there is
			// nothing to clone or change.
			res.Skipped = append(res.Skipped, Skipped{Repo: r.FullName(), Reason: "empty repository"})
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
	// Custom properties take a REST call per 100 repos, so they are only
	// fetched for campaigns that use them, and cached apart: a listing with
	// them also serves campaigns that don't.
	keys := []string{owner + "@properties"}
	if !rs.props {
		keys = append(keys, owner)
	}
	if rs.Cache != nil {
		for _, key := range keys {
			if repos, ok := rs.Cache.Get(key); ok {
				rs.progress(fmt.Sprintf("Listed %s: %s, from the cache", owner, countRepos(len(repos))))
				return repos, nil
			}
		}
	}
	rs.progress("Listing " + owner + "'s repos")
	repos, err := rs.Client.ListOwnerRepos(ctx, owner)
	if err != nil {
		return nil, err
	}
	rs.progress(fmt.Sprintf("Listed %s: %s", owner, countRepos(len(repos))))
	key := owner
	if rs.props {
		props, err := rs.Client.OwnerProperties(ctx, owner)
		if err != nil && rs.RequireProperties {
			return nil, err
		}
		for i := range repos {
			repos[i].Properties = props[repos[i].FullName()]
		}
		// A listing whose properties couldn't be read isn't cached as one
		// with them, so a later run asks again.
		if err == nil {
			key = owner + "@properties"
		}
	}
	if rs.Cache != nil {
		rs.Cache.Put(key, repos)
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

// progress reports a resolution step, when anyone is listening.
func (rs *Resolver) progress(msg string) {
	if rs.Progress != nil {
		rs.Progress(msg)
	}
}

// countRepos formats a count of repos.
func countRepos(n int) string {
	if n == 1 {
		return "1 repo"
	}
	return fmt.Sprintf("%d repos", n)
}

// candidates gathers the repos the campaign could target. When include
// filters apply to orgs, every org is listed. When include.repos is the
// whole selection, only what it names is fetched: an owner/name by itself,
// a bare name in each org, and a pattern from a listing of its owner, or
// of orgs for a bare or wildcard-owner pattern.
func (rs *Resolver) candidates(ctx context.Context, t campaign.Targets) (map[string]ghapi.Repo, error) {
	candidates := map[string]ghapi.Repo{}
	listed := map[string]bool{}
	list := func(owner string, keep func(ghapi.Repo) bool) error {
		if listed[strings.ToLower(owner)] {
			return nil
		}
		listed[strings.ToLower(owner)] = true
		repos, err := rs.listOwner(ctx, owner)
		if err != nil {
			return err
		}
		for _, r := range repos {
			if keep(r) {
				candidates[strings.ToLower(r.FullName())] = r
			}
		}
		return nil
	}
	named := func(r ghapi.Repo) bool { return campaign.MatchAny(t.Include.Repos, r.FullName()) != "" }
	if !OnlyRepos(t.Include) {
		for _, org := range t.Orgs {
			if err := list(org, func(ghapi.Repo) bool { return true }); err != nil {
				return nil, err
			}
		}
	}
	// Patterns first, so a name in an org they list needs no lookup.
	var exact, bare []string
	for _, name := range t.Include.Repos {
		owner, _, hasOwner := strings.Cut(name, "/")
		switch {
		case !hasOwner && !campaign.IsPattern(name):
			bare = append(bare, name)
		case !hasOwner || campaign.IsPattern(owner):
			// A bare or wildcard-owner pattern matches the repos of orgs;
			// load rejects one without orgs.
			for _, org := range t.Orgs {
				if err := list(org, named); err != nil {
					return nil, err
				}
			}
		case campaign.IsPattern(name):
			if err := list(owner, named); err != nil {
				return nil, err
			}
		default:
			exact = append(exact, name)
		}
	}
	var lookup []string
	required := map[string]bool{}
	for _, name := range exact {
		if _, ok := candidates[strings.ToLower(name)]; !ok {
			lookup = append(lookup, name)
			required[strings.ToLower(name)] = true
		}
	}
	// A bare name is that repo in each org, where it exists.
	for _, name := range bare {
		for _, org := range t.Orgs {
			if !listed[strings.ToLower(org)] {
				lookup = append(lookup, org+"/"+name)
			}
		}
	}
	found, err := rs.lookup(ctx, lookup)
	if err != nil {
		return nil, err
	}
	for _, name := range lookup {
		r, ok := found[strings.ToLower(name)]
		if !ok {
			if required[strings.ToLower(name)] {
				return nil, fmt.Errorf("%s: %w", name, ghapi.ErrNotFound)
			}
			continue
		}
		candidates[strings.ToLower(r.FullName())] = r
	}
	return candidates, nil
}

// named looks up only the repos in Only, keeping those the campaign could
// target: in one of its orgs, or named by include.repos. Filters still
// apply to them, so a run limited with --repo costs a batched lookup
// instead of listing and checking every org.
func (rs *Resolver) named(ctx context.Context, t campaign.Targets) (map[string]ghapi.Repo, error) {
	rs.progress("Looking up " + countRepos(len(rs.Only)))
	var keep []string
	for _, name := range rs.Only {
		owner, _, _ := strings.Cut(name, "/")
		inOrg := slices.ContainsFunc(t.Orgs, func(o string) bool { return strings.EqualFold(o, owner) })
		if inOrg || campaign.MatchAny(t.Include.Repos, name) != "" {
			keep = append(keep, name)
		}
		// Anything else is not a candidate; Limit reports it.
	}
	found, err := rs.lookup(ctx, keep)
	if err != nil {
		return nil, err
	}
	candidates := map[string]ghapi.Repo{}
	for _, name := range keep {
		r, ok := found[strings.ToLower(name)]
		if !ok {
			return nil, fmt.Errorf("%s: %w", name, ghapi.ErrNotFound)
		}
		candidates[strings.ToLower(r.FullName())] = r
	}
	return candidates, nil
}

// propertyConcurrency is how many repos' custom properties are read at
// once; each is a REST call.
const propertyConcurrency = 8

// lookup fetches repos by name in batched queries, with their custom
// properties when the campaign uses them. Missing repos are absent.
func (rs *Resolver) lookup(ctx context.Context, names []string) (map[string]ghapi.Repo, error) {
	if len(names) == 0 {
		return map[string]ghapi.Repo{}, nil
	}
	found, err := rs.Client.GetRepos(ctx, names)
	if err != nil {
		return nil, err
	}
	if !rs.props {
		return found, nil
	}
	keys := slices.Sorted(maps.Keys(found))
	props := make([]map[string]string, len(keys))
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(propertyConcurrency)
	for i, k := range keys {
		g.Go(func() error {
			p, err := rs.repoProperties(gctx, found[k].FullName())
			props[i] = p
			return err
		})
	}
	if err := g.Wait(); err != nil {
		return nil, err
	}
	for i, k := range keys {
		r := found[k]
		r.Properties = props[i]
		found[k] = r
	}
	return found, nil
}

// repoProperties returns one repo's custom properties. An error reading
// them only matters when the campaign requires them; otherwise the repo
// has none.
func (rs *Resolver) repoProperties(ctx context.Context, name string) (map[string]string, error) {
	props, err := rs.Client.RepoProperties(ctx, name)
	if err != nil && rs.RequireProperties {
		return nil, err
	}
	return props, nil
}
