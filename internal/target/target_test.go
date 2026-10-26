package target

import (
	"context"
	"errors"
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/rossedman/gh-fleet/internal/campaign"
	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/ghapi/fake"
)

// yes is a pointer to true.
var yes = func() *bool { b := true; return &b }()

// no is a pointer to false.
var no = func() *bool { b := false; return &b }()

func TestEvaluate(t *testing.T) {
	t.Parallel()
	svc := ghapi.Repo{Owner: "acme", Name: "api", Topics: []string{"service"}, Language: "Go"}
	tests := []struct {
		name    string
		targets campaign.Targets
		repo    ghapi.Repo
		facts   Facts
		match   bool
		reasons []string
		exclude string
	}{
		{name: "org only", repo: svc, match: true, reasons: []string{"org:acme"}},
		{name: "topic or", targets: campaign.Targets{Include: campaign.Include{Topics: []string{"lib", "SERVICE"}}}, repo: svc, match: true, reasons: []string{"topic:SERVICE"}},
		{name: "topic miss", targets: campaign.Targets{Include: campaign.Include{Topics: []string{"lib"}}}, repo: svc},
		{name: "language and topic", targets: campaign.Targets{Include: campaign.Include{Topics: []string{"service"}, Languages: []string{"go"}}}, repo: svc, match: true, reasons: []string{"topic:service", "language:Go"}},
		{name: "language miss", targets: campaign.Targets{Include: campaign.Include{Languages: []string{"Rust"}}}, repo: svc},
		{name: "no language", targets: campaign.Targets{Include: campaign.Include{Languages: []string{"Go"}}}, repo: ghapi.Repo{Owner: "a", Name: "b"}},
		{name: "files", targets: campaign.Targets{Include: campaign.Include{Files: []string{"Dockerfile", "go.mod"}}}, repo: svc, facts: Facts{Files: []string{"go.mod"}}, match: true, reasons: []string{"file:go.mod"}},
		{name: "files miss", targets: campaign.Targets{Include: campaign.Include{Files: []string{"Dockerfile"}}}, repo: svc},
		{name: "search", targets: campaign.Targets{Include: campaign.Include{Search: "x"}}, repo: svc, facts: Facts{SearchHit: true}, match: true, reasons: []string{"search"}},
		{name: "search miss", targets: campaign.Targets{Include: campaign.Include{Search: "x"}}, repo: svc},
		{name: "explicit bypasses filters", targets: campaign.Targets{Include: campaign.Include{Topics: []string{"lib"}, Repos: []string{"ACME/api"}}}, repo: svc, match: true, reasons: []string{"include.repos"}},
		{name: "exclude wins over explicit", targets: campaign.Targets{Include: campaign.Include{Repos: []string{"acme/api"}}, Exclude: campaign.Exclude{Repos: []string{"acme/api"}}}, repo: svc, exclude: "exclude.repos"},
		{name: "name pattern", targets: campaign.Targets{Include: campaign.Include{Names: []string{"web", "a*"}}}, repo: svc, match: true, reasons: []string{"name:a*"}},
		{name: "name pattern and topic", targets: campaign.Targets{Include: campaign.Include{Names: []string{"acme/a*"}, Topics: []string{"service"}}}, repo: svc, match: true, reasons: []string{"name:acme/a*", "topic:service"}},
		{name: "name pattern miss", targets: campaign.Targets{Include: campaign.Include{Names: []string{"b*"}}}, repo: svc},
		{name: "explicit pattern bypasses filters", targets: campaign.Targets{Include: campaign.Include{Topics: []string{"lib"}, Repos: []string{"acme/a?i"}}}, repo: svc, match: true, reasons: []string{"include.repos"}},
		{name: "excluded by pattern", targets: campaign.Targets{Exclude: campaign.Exclude{Repos: []string{"*/ap*"}}}, repo: svc, exclude: "exclude.repos"},
		{name: "archived default", repo: ghapi.Repo{Owner: "a", Name: "b", Archived: true}, exclude: "archived"},
		{name: "archived allowed", targets: campaign.Targets{Exclude: campaign.Exclude{Archived: no}}, repo: ghapi.Repo{Owner: "a", Name: "b", Archived: true}, match: true, reasons: []string{"org:a"}},
		{name: "fork", targets: campaign.Targets{Exclude: campaign.Exclude{Forks: yes}}, repo: ghapi.Repo{Owner: "a", Name: "b", Fork: true}, exclude: "fork"},
		{name: "excluded topic", targets: campaign.Targets{Exclude: campaign.Exclude{Topics: []string{"no-fleet"}}}, repo: ghapi.Repo{Owner: "a", Name: "b", Topics: []string{"no-fleet"}}, exclude: "topic:no-fleet"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			d := Evaluate(tt.targets, tt.repo, tt.facts)
			if d.Match != tt.match || !slices.Equal(d.Reasons, tt.reasons) || d.Excluded != tt.exclude {
				t.Fatalf("Evaluate = %+v", d)
			}
		})
	}
}

// newFake returns a fake org with a mix of repos.
func newFake() *fake.Client {
	f := fake.New()
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "api", Topics: []string{"service"}, Language: "Go", CanPush: true})
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "web", Topics: []string{"service"}, Language: "TypeScript", CanPush: true})
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "lib", Language: "Go", CanPush: true})
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "ro", Topics: []string{"service"}, Language: "Go"})
	f.AddRepo(ghapi.Repo{Owner: "acme", Name: "sandbox", Topics: []string{"service"}, Language: "Go", CanPush: true})
	f.AddRepo(ghapi.Repo{Owner: "other", Name: "legacy", CanPush: true})
	f.Files["acme/api"] = []string{".github/workflows"}
	f.Files["acme/ro"] = []string{".github/workflows"}
	f.Files["acme/sandbox"] = []string{".github/workflows"}
	f.Search["uses: actions/"] = []string{"acme/api", "acme/ro", "acme/web"}
	return f
}

func TestResolve(t *testing.T) {
	t.Parallel()
	f := newFake()
	rs := &Resolver{Client: f, Cache: &Cache{Dir: t.TempDir()}}
	res, err := rs.Resolve(context.Background(), campaign.Targets{
		Orgs: []string{"acme"},
		Include: campaign.Include{
			Topics: []string{"service"}, Files: []string{".github/workflows"}, Search: "uses: actions/",
			Repos: []string{"other/legacy"},
		},
		Exclude: campaign.Exclude{Repos: []string{"acme/sandbox"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Names(); !slices.Equal(got, []string{"acme/api", "other/legacy"}) {
		t.Fatalf("targets = %v", got)
	}
	if res.Targets[0].Reasons[0] != "topic:service" || len(res.Targets[0].Reasons) != 3 {
		t.Fatalf("reasons = %v", res.Targets[0].Reasons)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].Repo != "acme/ro" || res.Skipped[0].Reason != "no push access" {
		t.Fatalf("skipped = %+v", res.Skipped)
	}
	if err := res.Limit([]string{"ACME/api"}); err != nil || len(res.Targets) != 1 {
		t.Fatalf("limit: %v", err)
	}
	if err := res.Limit([]string{"acme/zzz", "acme/aaa"}); err == nil || !strings.Contains(err.Error(), "acme/aaa, acme/zzz") {
		t.Fatalf("limit err = %v", err)
	}
	if err := res.Limit(nil); err != nil {
		t.Fatal(err)
	}

	// Read-only resolution keeps repos without push access.
	rs.AllowReadOnly = true
	res, err = rs.Resolve(context.Background(), campaign.Targets{Orgs: []string{"acme"}, Include: campaign.Include{Search: "uses: actions/"}})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Names(); !slices.Equal(got, []string{"acme/api", "acme/ro", "acme/web"}) {
		t.Fatalf("read-only targets = %v", got)
	}
	if f.CallCount("ListOwnerRepos") != 1 {
		t.Fatal("org listing not cached")
	}
}

func TestResolveErrors(t *testing.T) {
	t.Parallel()
	for _, method := range []string{"ListOwnerRepos", "GetRepo", "PathsExist", "SearchCode"} {
		f := newFake()
		f.Errors[method] = errors.New("boom")
		rs := &Resolver{Client: f}
		_, err := rs.Resolve(context.Background(), campaign.Targets{
			Orgs:    []string{"acme"},
			Include: campaign.Include{Files: []string{"x"}, Search: "uses: actions/", Repos: []string{"other/legacy"}},
		})
		if err == nil {
			t.Errorf("%s error not returned", method)
		}
	}
	// Search without orgs searches everywhere.
	f := newFake()
	res, err := (&Resolver{Client: f}).Resolve(context.Background(), campaign.Targets{Include: campaign.Include{Search: "uses: actions/", Repos: []string{"acme/api"}}})
	if err != nil || len(res.Targets) != 1 {
		t.Fatalf("res = %+v, %v", res, err)
	}
}

func TestCache(t *testing.T) {
	t.Parallel()
	now := time.Unix(1000, 0)
	c := &Cache{Dir: t.TempDir(), Now: func() time.Time { return now }}
	if _, ok := c.Get("acme"); ok {
		t.Fatal("empty cache hit")
	}
	c.Put("acme", []ghapi.Repo{{Owner: "acme", Name: "api"}})
	if repos, ok := c.Get("ACME"); !ok || repos[0].Name != "api" {
		t.Fatal("cache miss")
	}
	now = now.Add(DefaultCacheTTL + time.Second)
	if _, ok := c.Get("acme"); ok {
		t.Fatal("stale entry returned")
	}
	if (&Cache{}).now().IsZero() {
		t.Fatal("default clock")
	}
	// Corrupt entries and unwritable dirs are ignored.
	bad := &Cache{Dir: t.TempDir()}
	bad.Put("x", nil)
	if err := writeFile(bad.path("x"), "{"); err != nil {
		t.Fatal(err)
	}
	if _, ok := bad.Get("x"); ok {
		t.Fatal("corrupt entry returned")
	}
	(&Cache{Dir: "/dev/null/nope"}).Put("x", nil)
}

// syntheticOrg builds a fake org of n repos with varied metadata.
func syntheticOrg(n int) *fake.Client {
	f := fake.New()
	langs := []string{"Go", "Python", "TypeScript", "Java", "Rust"}
	for i := range n {
		r := ghapi.Repo{
			Owner: "big", Name: fmt.Sprintf("repo-%04d", i), Language: langs[i%len(langs)],
			Archived: i%50 == 0, Fork: i%40 == 0, CanPush: i%30 != 0,
		}
		if i%3 == 0 {
			r.Topics = []string{"service"}
		}
		if i%7 == 0 {
			r.Topics = append(r.Topics, "no-fleet")
		}
		f.AddRepo(r)
		if i%2 == 0 {
			f.Files[r.FullName()] = []string{".github/workflows"}
		}
	}
	return f
}

func BenchmarkResolve5000(b *testing.B) {
	f := syntheticOrg(5000)
	rs := &Resolver{Client: f}
	targets := campaign.Targets{
		Orgs:    []string{"big"},
		Include: campaign.Include{Topics: []string{"service"}, Languages: []string{"Go", "Python"}, Files: []string{".github/workflows"}},
		Exclude: campaign.Exclude{Topics: []string{"no-fleet"}},
	}
	b.ReportAllocs()
	for b.Loop() {
		res, err := rs.Resolve(context.Background(), targets)
		if err != nil || len(res.Targets) == 0 {
			b.Fatal(err)
		}
	}
}

func BenchmarkEvaluate(b *testing.B) {
	t := campaign.Targets{
		Include: campaign.Include{Topics: []string{"a", "b", "service"}, Languages: []string{"Go"}, Files: []string{"x", "y"}},
		Exclude: campaign.Exclude{Topics: []string{"no-fleet"}, Repos: []string{"o/skip"}},
	}
	r := ghapi.Repo{Owner: "o", Name: "r", Topics: []string{"service", "z"}, Language: "Go"}
	facts := Facts{Files: []string{"y"}}
	b.ReportAllocs()
	for b.Loop() {
		if !Evaluate(t, r, facts).Match {
			b.Fatal("no match")
		}
	}
}

// splitList turns a comma-separated fuzz input into a list.
func splitList(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}

func FuzzEvaluate(f *testing.F) {
	f.Add("service", "Go", "go.mod", "", "acme/api", "no-fleet", "service,lib", "Go", "go.mod", false, false, false)
	f.Add("", "", "", "x", "", "", "", "", "", true, true, true)
	f.Add("a,b", "rust", "Dockerfile", "", "acme/api", "acme/api", "B", "", "", false, false, true)
	f.Fuzz(func(t *testing.T, topics, langs, files, search, repos, exTopics, repoTopics, repoLang, repoFiles string, archived, fork, hit bool) {
		tg := campaign.Targets{
			Include: campaign.Include{Topics: splitList(topics), Languages: splitList(langs), Files: splitList(files), Search: search, Repos: splitList(repos)},
			Exclude: campaign.Exclude{Topics: splitList(exTopics), Repos: splitList(exTopics)},
		}
		r := ghapi.Repo{Owner: "acme", Name: "api", Topics: splitList(repoTopics), Language: repoLang, Archived: archived, Fork: fork}
		d := Evaluate(tg, r, Facts{Files: splitList(repoFiles), SearchHit: hit})

		// Exclusion always wins and never comes with a match.
		if d.Excluded != "" && d.Match {
			t.Fatal("excluded repo matched")
		}
		if (archived || fork) && d.Match {
			t.Fatal("archived or fork matched with default excludes")
		}
		if d.Match && len(d.Reasons) == 0 {
			t.Fatal("match without reasons")
		}
		if !d.Match && len(d.Reasons) > 0 {
			t.Fatal("reasons without match")
		}
		// An explicit repo always matches unless excluded.
		if containsFold(tg.Include.Repos, r.FullName()) && d.Excluded == "" && !d.Match {
			t.Fatal("include.repos did not bypass filters")
		}
		// A filter match names every include key that was set.
		if d.Match && d.Reasons[0] != "include.repos" {
			want := 0
			for _, set := range []bool{len(tg.Include.Topics) > 0, len(tg.Include.Languages) > 0, len(tg.Include.Files) > 0, tg.Include.Search != ""} {
				if set {
					want++
				}
			}
			if want == 0 {
				want = 1 // the org reason
			}
			if len(d.Reasons) != want {
				t.Fatalf("reasons %v, want %d", d.Reasons, want)
			}
		}
		// MetadataOnly never rules out a repo Evaluate would match.
		if d.Match && !MetadataOnly(tg, r) {
			t.Fatal("MetadataOnly dropped a matching repo")
		}
	})
}

// writeFile overwrites a file, for corrupting cache entries.
func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o600)
}

func TestResolvePatterns(t *testing.T) {
	t.Parallel()
	f := newFake()
	f.AddRepo(ghapi.Repo{Owner: "other", Name: "tool-a", CanPush: true})
	f.AddRepo(ghapi.Repo{Owner: "other", Name: "tool-b", CanPush: true})
	rs := &Resolver{Client: f}
	res, err := rs.Resolve(context.Background(), campaign.Targets{
		Orgs: []string{"acme"},
		Include: campaign.Include{
			Names: []string{"a*", "w*"},
			// other isn't an org: listed once for both patterns, keeping
			// only the matches. */x is matched against the org's repos.
			Repos: []string{"other/tool-*", "OTHER/legacy", "*/lib", "other/t*"},
		},
		Exclude: campaign.Exclude{Repos: []string{"*/tool-b"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Names(); !slices.Equal(got, []string{"acme/api", "acme/lib", "acme/web", "other/legacy", "other/tool-a"}) {
		t.Fatalf("targets = %v", got)
	}
	if n := f.CallCount("ListOwnerRepos"); n != 2 {
		t.Errorf("ListOwnerRepos called %d times, want 2", n)
	}
	f.Errors["ListOwnerRepos"] = errors.New("boom")
	if _, err := (&Resolver{Client: f}).Resolve(context.Background(), campaign.Targets{Include: campaign.Include{Repos: []string{"other/tool-*"}}}); err == nil {
		t.Error("listing error swallowed")
	}
}

func TestResolveProgress(t *testing.T) {
	t.Parallel()
	var msgs []string
	rs := &Resolver{Client: newFake(), Cache: &Cache{Dir: t.TempDir()}, Progress: func(m string) { msgs = append(msgs, m) }}
	targets := campaign.Targets{Orgs: []string{"acme"}, Include: campaign.Include{Files: []string{".github/workflows", "go.mod"}, Search: "uses: actions/"}}
	for range 2 {
		if _, err := rs.Resolve(context.Background(), targets); err != nil {
			t.Fatal(err)
		}
	}
	want := []string{
		"Listing acme's repos",
		"Listed acme: 5 repos",
		"Checking .github/workflows, go.mod in 5 repos",
		`Searching code for "uses: actions/"`,
		"Listed acme: 5 repos, from the cache",
		"Checking .github/workflows, go.mod in 5 repos",
		`Searching code for "uses: actions/"`,
	}
	if !slices.Equal(msgs, want) {
		t.Fatalf("progress =\n%s\nwant\n%s", strings.Join(msgs, "\n"), strings.Join(want, "\n"))
	}
	if countRepos(1) != "1 repo" {
		t.Error("repos")
	}
}
