package ghapi

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// apiServer is a scripted GitHub API: GraphQL requests are answered by
// gql, REST requests by the rest routes, and every request is recorded.
type apiServer struct {
	mu       sync.Mutex
	gql      func(query string, vars map[string]any) (any, int)
	rest     map[string]func(r *http.Request, body []byte) (any, int)
	requests []string
	bodies   []string
}

// ServeHTTP dispatches a request.
func (s *apiServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var body []byte
	if r.Body != nil {
		body, _ = io.ReadAll(r.Body)
	}
	s.mu.Lock()
	s.requests = append(s.requests, r.Method+" "+r.URL.RequestURI())
	s.bodies = append(s.bodies, string(body))
	s.mu.Unlock()
	w.Header().Set("Content-Type", "application/json")
	if r.URL.Path == "/graphql" {
		var req struct {
			Query     string         `json:"query"`
			Variables map[string]any `json:"variables"`
		}
		_ = json.Unmarshal(body, &req)
		data, code := s.gql(req.Query, req.Variables)
		w.WriteHeader(code)
		resp := map[string]any{"data": data}
		// A "__errors" key in the data becomes the response's GraphQL
		// errors, as GitHub sends them alongside partial data.
		if m, ok := data.(map[string]any); ok && m["__errors"] != nil {
			errs := m["__errors"]
			delete(m, "__errors")
			resp = map[string]any{"data": m, "errors": errs}
		}
		_ = json.NewEncoder(w).Encode(resp)
		return
	}
	key := r.Method + " " + r.URL.Path
	h, ok := s.rest[key]
	if !ok {
		w.WriteHeader(http.StatusNotFound)
		_, _ = io.WriteString(w, `{"message":"Not Found"}`)
		return
	}
	v, code := h(r, body)
	w.WriteHeader(code)
	if v != nil {
		_ = json.NewEncoder(w).Encode(v)
	}
}

// newClient returns a GitHub client talking to s.
func newClient(t *testing.T, s *apiServer) *GitHub {
	t.Helper()
	c, err := New(Options{Host: "github.com", AuthToken: "test-token", WriteRate: 1000, Base: handlerTransport{s}})
	if err != nil {
		t.Fatal(err)
	}
	return c
}

// repoNode builds a GraphQL repository node.
func repoNode(owner, name string, perm string, topics ...string) map[string]any {
	nodes := []any{}
	for _, tp := range topics {
		nodes = append(nodes, map[string]any{"topic": map[string]any{"name": tp}})
	}
	return map[string]any{
		"name": name, "owner": map[string]any{"login": owner}, "url": "https://github.com/" + owner + "/" + name,
		"isArchived": false, "isFork": false, "viewerPermission": perm,
		"defaultBranchRef": map[string]any{"name": "main"}, "primaryLanguage": map[string]any{"name": "Go"},
		"repositoryTopics": map[string]any{"nodes": nodes},
	}
}

func TestListOwnerReposPaginates(t *testing.T) {
	t.Parallel()
	s := &apiServer{gql: func(q string, vars map[string]any) (any, int) {
		page := map[string]any{"hasNextPage": true, "endCursor": "c1"}
		nodes := []any{repoNode("acme", "api", "WRITE", "service")}
		if vars["after"] == "c1" {
			page = map[string]any{"hasNextPage": false, "endCursor": ""}
			nodes = []any{repoNode("acme", "web", "READ")}
		}
		return map[string]any{"repositoryOwner": map[string]any{"repositories": map[string]any{"pageInfo": page, "nodes": nodes}}}, 200
	}}
	repos, err := newClient(t, s).ListOwnerRepos(context.Background(), "acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(repos) != 2 || repos[0].FullName() != "acme/api" || !repos[0].CanPush || repos[1].CanPush ||
		repos[0].Topics[0] != "service" || repos[0].Language != "Go" || repos[0].DefaultBranch != "main" {
		t.Fatalf("repos = %+v", repos)
	}

	missing := &apiServer{gql: func(string, map[string]any) (any, int) { return map[string]any{"repositoryOwner": nil}, 200 }}
	if _, err := newClient(t, missing).ListOwnerRepos(context.Background(), "nope"); err == nil {
		t.Fatal("expected error for missing owner")
	}
	broken := &apiServer{gql: func(string, map[string]any) (any, int) { return nil, 502 }}
	c := newClient(t, broken)
	c.Transport.MaxAttempts = 1
	if _, err := c.ListOwnerRepos(context.Background(), "acme"); err == nil {
		t.Fatal("expected HTTP error")
	}
}

func TestGetRepos(t *testing.T) {
	t.Parallel()
	alias := regexp.MustCompile(`(r\d+): repository\(owner: "([^"]+)", name: "([^"]+)"\)`)
	s := &apiServer{gql: func(q string, _ map[string]any) (any, int) {
		data := map[string]any{}
		var errs []any
		for _, m := range alias.FindAllStringSubmatch(q, -1) {
			switch m[3] {
			case "gone":
				data[m[1]] = nil
			case "hidden":
				data[m[1]] = nil
				errs = append(errs, map[string]any{"type": "NOT_FOUND", "path": []any{m[1]}, "message": "Could not resolve to a Repository"})
			case "broken":
				errs = append(errs, map[string]any{"type": "FORBIDDEN", "path": []any{m[1]}, "message": "no"})
			default:
				data[m[1]] = repoNode(m[2], m[3], "ADMIN")
			}
		}
		if errs != nil {
			data["__errors"] = errs
		}
		return data, 200
	}}
	c := newClient(t, s)
	names := []string{"Acme/API", "acme/gone", "acme/hidden"}
	for i := range BatchSize {
		names = append(names, fmt.Sprintf("acme/r%d", i))
	}
	got, err := c.GetRepos(context.Background(), names)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != BatchSize+1 || !got["acme/api"].CanPush || got["acme/r49"].Name != "r49" {
		t.Fatalf("got %d repos: %v", len(got), got["acme/api"])
	}
	if _, ok := got["acme/gone"]; ok {
		t.Error("missing repo returned")
	}
	if n := len(s.requests); n != 2 {
		t.Errorf("%d requests for %d repos, want 2", n, len(names))
	}

	if _, err := c.GetRepos(context.Background(), []string{"acme/api", "acme/broken"}); err == nil {
		t.Error("an error other than not found was ignored")
	}
	if _, err := c.GetRepos(context.Background(), []string{"bad"}); err == nil {
		t.Error("expected invalid name error")
	}
	if onlyNotFound(errors.New("x")) {
		t.Error("onlyNotFound on a plain error")
	}
}

func TestPathsExistBatches(t *testing.T) {
	t.Parallel()
	var mu sync.Mutex
	var queries []string
	s := &apiServer{gql: func(q string, _ map[string]any) (any, int) {
		mu.Lock()
		queries = append(queries, q)
		mu.Unlock()
		if strings.Contains(q, `"boom"`) {
			return nil, 400
		}
		out := map[string]any{}
		for i := range BatchSize {
			alias := fmt.Sprintf("r%d", i)
			if !strings.Contains(q, alias+":") {
				continue
			}
			// Every even repo has the first path.
			objs := map[string]any{"p0": nil, "p1": nil}
			if i%2 == 0 {
				objs["p0"] = map[string]any{"__typename": "Tree"}
			}
			out[alias] = objs
		}
		return out, 200
	}}
	var repos []Repo
	for i := range 120 {
		repos = append(repos, Repo{Owner: "acme", Name: fmt.Sprintf("r%03d", i), DefaultBranch: "main"})
	}
	repos[1].DefaultBranch = ""
	got, err := newClient(t, s).PathsExist(context.Background(), repos, []string{".github/workflows", "/Dockerfile"})
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 3 {
		t.Fatalf("want 3 batched queries, got %d", len(queries))
	}
	// Batches run concurrently, so find the first by its first repo.
	first := queries[slices.IndexFunc(queries, func(q string) bool { return strings.Contains(q, `"r000"`) })]
	if !strings.Contains(first, `object(expression: "main:.github/workflows")`) || !strings.Contains(first, `"HEAD:Dockerfile"`) {
		t.Fatalf("query = %s", first[:300])
	}
	if len(got["acme/r000"]) != 1 || got["acme/r001"] != nil || len(got["acme/r050"]) != 1 {
		t.Fatalf("got = %v", got)
	}
	if m, err := newClient(t, s).PathsExist(context.Background(), repos, nil); err != nil || len(m) != 0 {
		t.Fatal("no paths should mean no queries")
	}
	// A failing batch fails the check.
	repos[70].Name = "boom"
	c := newClient(t, s)
	c.Transport.MaxAttempts = 1
	if _, err := c.PathsExist(context.Background(), repos, []string{"go.mod"}); err == nil {
		t.Error("batch error swallowed")
	}
}

func TestSearchCodePaginates(t *testing.T) {
	t.Parallel()
	s := &apiServer{rest: map[string]func(*http.Request, []byte) (any, int){
		"GET /search/code": func(r *http.Request, _ []byte) (any, int) {
			if !strings.Contains(r.URL.Query().Get("q"), "org:acme") {
				return map[string]any{"message": "bad query"}, 422
			}
			items := []any{}
			n := 100
			if r.URL.Query().Get("page") == "2" {
				n = 3
			}
			for i := range n {
				items = append(items, map[string]any{"repository": map[string]any{"full_name": fmt.Sprintf("acme/r%d", i%5)}})
			}
			return map[string]any{"total_count": 103, "items": items}, 200
		},
	}}
	c := newClient(t, s)
	res, err := c.SearchCode(context.Background(), "uses: actions/ org:acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repos) != 5 || res.Repos[0] != "acme/r0" || res.Partial {
		t.Fatalf("result = %+v", res)
	}
	if _, err := c.SearchCode(context.Background(), "x"); err == nil {
		t.Fatal("expected error")
	}
}

// searchServer answers code searches over files, as GitHub does: a
// size:lo..hi qualifier filters by file size, and only the first
// codeSearchCap matches can be paged through.
func searchServer(files []searchFile, incomplete bool) *apiServer {
	sizeQ := regexp.MustCompile(`size:(\d+)\.\.(\d+)`)
	return &apiServer{rest: map[string]func(*http.Request, []byte) (any, int){
		"GET /search/code": func(r *http.Request, _ []byte) (any, int) {
			lo, hi := 0, maxIndexedFileSize
			if m := sizeQ.FindStringSubmatch(r.URL.Query().Get("q")); m != nil {
				lo, _ = strconv.Atoi(m[1])
				hi, _ = strconv.Atoi(m[2])
			}
			var match []searchFile
			for _, f := range files {
				if f.size >= lo && f.size <= hi {
					match = append(match, f)
				}
			}
			page, _ := strconv.Atoi(r.URL.Query().Get("page"))
			items := []any{}
			for i := (page - 1) * 100; i < min(page*100, len(match), codeSearchCap); i++ {
				items = append(items, map[string]any{"repository": map[string]any{"full_name": match[i].repo}})
			}
			return map[string]any{"total_count": len(match), "incomplete_results": incomplete, "items": items}, 200
		},
	}}
}

// searchFile is one indexed file in searchServer.
type searchFile struct {
	repo string
	size int
}

func TestSearchCodeSplitsPastTheCap(t *testing.T) {
	t.Parallel()
	// 2500 matching files in 40 busy repos, and one file in a quiet repo
	// that a single query's first 1000 results never reach.
	var files []searchFile
	for i := range 2500 {
		files = append(files, searchFile{fmt.Sprintf("acme/busy%d", i%40), 100 + i*50})
	}
	files = append(files, searchFile{"acme/quiet", 300000})
	s := searchServer(files, false)
	res, err := newClient(t, s).SearchCode(context.Background(), "uses: actions/ org:acme")
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Repos) != 41 || !slices.Contains(res.Repos, "acme/quiet") || res.Partial {
		t.Fatalf("got %d repos, partial %v, quiet found %v", len(res.Repos), res.Partial, slices.Contains(res.Repos, "acme/quiet"))
	}

	// More matches than the cap in one file size, or a query that sets
	// its own size, can't be split: the result says it is partial.
	same := make([]searchFile, 1200)
	for i := range same {
		same[i] = searchFile{fmt.Sprintf("acme/r%d", i), 10}
	}
	for _, q := range []string{"x org:acme", "x size:>5 org:acme"} {
		res, err := newClient(t, searchServer(same, false)).SearchCode(context.Background(), q)
		if err != nil || !res.Partial || len(res.Repos) != codeSearchCap {
			t.Errorf("%s: partial %v, %d repos, %v", q, res.Partial, len(res.Repos), err)
		}
	}
	// GitHub's own incomplete_results flag makes it partial too.
	res, err = newClient(t, searchServer(files[:5], true)).SearchCode(context.Background(), "x")
	if err != nil || !res.Partial {
		t.Errorf("incomplete results: %+v, %v", res, err)
	}
	// An error in a split half fails the search.
	broken := searchServer(files, false)
	ok := broken.rest["GET /search/code"]
	broken.rest["GET /search/code"] = func(r *http.Request, b []byte) (any, int) {
		if strings.Contains(r.URL.Query().Get("q"), "size:") {
			return map[string]any{"message": "boom"}, 422
		}
		return ok(r, b)
	}
	if _, err := newClient(t, broken).SearchCode(context.Background(), "x"); err == nil {
		t.Error("error in a split half was swallowed")
	}
}

func TestSplitSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct{ lo, hi, want int }{
		{0, maxIndexedFileSize, 626},
		{0, 626, 24},
		{5, 6, 5},
		{0, 1, 0},
	} {
		if got := splitSize(tc.lo, tc.hi); got != tc.want {
			t.Errorf("splitSize(%d, %d) = %d, want %d", tc.lo, tc.hi, got, tc.want)
		}
	}
}

func TestPRsByHeadAndSearch(t *testing.T) {
	t.Parallel()
	s := &apiServer{
		gql: func(q string, _ map[string]any) (any, int) {
			return map[string]any{
				"r0": map[string]any{"pullRequests": map[string]any{"nodes": []any{map[string]any{
					"number": 7, "url": "https://github.com/acme/api/pull/7", "state": "OPEN", "title": "t", "body": "b",
					"isDraft": false, "headRefName": "fleet/x", "headRefOid": "abc", "baseRefName": "main",
					"reviewDecision": "APPROVED", "labels": map[string]any{"nodes": []any{map[string]any{"name": "fleet"}}},
					"id": "PR_kw", "mergeStateStatus": "CLEAN", "autoMergeRequest": map[string]any{"enabledAt": "2026-01-01T00:00:00Z"},
					"commits": map[string]any{"nodes": []any{map[string]any{"commit": map[string]any{"statusCheckRollup": map[string]any{"state": "SUCCESS"}}}}},
				}}}},
				"r1": map[string]any{"pullRequests": map[string]any{"nodes": []any{}}},
			}, 200
		},
		rest: map[string]func(*http.Request, []byte) (any, int){
			"GET /search/issues": func(r *http.Request, _ []byte) (any, int) {
				q := r.URL.Query().Get("q")
				if !strings.Contains(q, `"gh-fleet:campaign=x"`) || !strings.Contains(q, "user:acme") {
					return map[string]any{"message": "bad"}, 422
				}
				merged := "2024-01-01T00:00:00Z"
				items := []any{
					map[string]any{"number": 1, "state": "open", "body": "<!-- gh-fleet:campaign=x -->", "repository_url": "https://api.github.com/repos/acme/a", "pull_request": map[string]any{}},
					map[string]any{"number": 2, "state": "closed", "body": "<!-- gh-fleet:campaign=x -->", "repository_url": "https://api.github.com/repos/acme/b", "pull_request": map[string]any{"merged_at": merged}},
					map[string]any{"number": 3, "state": "closed", "body": "<!-- gh-fleet:campaign=x -->", "repository_url": "https://api.github.com/repos/acme/c", "pull_request": map[string]any{}},
					map[string]any{"number": 4, "state": "open", "body": "mentions gh-fleet:campaign=x only", "repository_url": "https://api.github.com/repos/acme/d", "pull_request": map[string]any{}},
				}
				return map[string]any{"total_count": len(items), "items": items}, 200
			},
		},
	}
	c := newClient(t, s)
	got, err := c.PRsByHead(context.Background(), []string{"acme/api", "acme/web"}, "fleet/x")
	if err != nil {
		t.Fatal(err)
	}
	p, ok := got["acme/api"]
	if !ok || len(got) != 1 || p.Number != 7 || p.Checks != "SUCCESS" || p.Review != "APPROVED" || p.Labels[0] != "fleet" || p.HeadSHA != "abc" ||
		p.NodeID != "PR_kw" || p.MergeState != "CLEAN" || !p.AutoMerge {
		t.Fatalf("got = %+v", got)
	}
	if _, err := c.PRsByHead(context.Background(), []string{"bad"}, "x"); err == nil {
		t.Fatal("expected invalid repo error")
	}

	prs, err := c.SearchCampaignPRs(context.Background(), []string{"user:acme"}, "<!-- gh-fleet:campaign=x -->")
	if err != nil {
		t.Fatal(err)
	}
	// Open and closed searches return the same items; duplicates and
	// non-exact marker matches are dropped.
	if len(prs) != 3 || prs[0].State != StateOpen || prs[1].State != StateMerged || prs[2].State != StateClosed || prs[0].Repo != "acme/a" {
		t.Fatalf("prs = %+v", prs)
	}
	if _, err := c.SearchCampaignPRs(context.Background(), nil, "<!-- gh-fleet:campaign=x -->"); err == nil {
		t.Fatal("expected search error")
	}
}

func TestWrites(t *testing.T) {
	t.Parallel()
	s := &apiServer{rest: map[string]func(*http.Request, []byte) (any, int){
		"POST /repos/acme/api/pulls": func(_ *http.Request, body []byte) (any, int) {
			var in map[string]any
			_ = json.Unmarshal(body, &in)
			return map[string]any{"number": 5, "html_url": "u", "state": "open", "title": in["title"], "head": map[string]any{"ref": in["head"], "sha": "s"}, "base": map[string]any{"ref": in["base"]}, "labels": []any{map[string]any{"name": "l"}}}, 201
		},
		"PATCH /repos/acme/api/pulls/5":                    func(*http.Request, []byte) (any, int) { return map[string]any{}, 200 },
		"POST /repos/acme/api/issues/5/labels":             func(*http.Request, []byte) (any, int) { return []any{}, 200 },
		"POST /repos/acme/api/pulls/5/requested_reviewers": func(*http.Request, []byte) (any, int) { return map[string]any{}, 201 },
		"POST /repos/acme/api/issues/5/comments":           func(*http.Request, []byte) (any, int) { return map[string]any{}, 201 },
		"DELETE /repos/acme/api/git/refs/heads/fleet/x":    func(*http.Request, []byte) (any, int) { return nil, 204 },
		"PUT /repos/acme/api/pulls/5/merge":                func(*http.Request, []byte) (any, int) { return map[string]any{"merged": true}, 200 },
		"PUT /repos/acme/api/pulls/5/update-branch":        func(*http.Request, []byte) (any, int) { return map[string]any{}, 202 },
		"DELETE /repos/acme/api/git/refs/heads/gone": func(*http.Request, []byte) (any, int) {
			return map[string]any{"message": "Reference does not exist"}, 422
		},
		"GET /rate_limit": func(*http.Request, []byte) (any, int) {
			return map[string]any{"resources": map[string]any{
				"core":    map[string]any{"limit": 5000, "remaining": 4000, "reset": 1700000000},
				"graphql": map[string]any{"limit": 5000, "remaining": 4999, "reset": 1700000000},
			}}, 200
		},
	}}
	c := newClient(t, s)
	ctx := context.Background()
	pr, err := c.CreatePR(ctx, "acme/api", NewPR{Title: "t", Body: "b", Head: "fleet/x", Base: "main", Draft: true})
	if err != nil || pr.Number != 5 || pr.State != StateOpen || pr.HeadRef != "fleet/x" || pr.Labels[0] != "l" {
		t.Fatalf("pr = %+v, %v", pr, err)
	}
	title, closed := "new", "closed"
	if err := c.UpdatePR(ctx, "acme/api", 5, PRUpdate{Title: &title, Body: &title, State: &closed}); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdatePR(ctx, "acme/api", 5, PRUpdate{}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddLabels(ctx, "acme/api", 5, []string{"fleet"}); err != nil {
		t.Fatal(err)
	}
	if err := c.AddLabels(ctx, "acme/api", 5, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.RequestReviewers(ctx, "acme/api", 5, []string{"octocat", "acme/platform"}); err != nil {
		t.Fatal(err)
	}
	if err := c.RequestReviewers(ctx, "acme/api", 5, nil); err != nil {
		t.Fatal(err)
	}
	if err := c.Comment(ctx, "acme/api", 5, "hi"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteBranch(ctx, "acme/api", "fleet/x"); err != nil {
		t.Fatal(err)
	}
	if err := c.DeleteBranch(ctx, "acme/api", "gone"); err != nil {
		t.Fatalf("missing branch should not fail: %v", err)
	}
	if err := c.MergePR(ctx, "acme/api", 5, MergeMethodSquash, "abc"); err != nil {
		t.Fatal(err)
	}
	if err := c.UpdateBranch(ctx, "acme/api", 5, "abc"); err != nil {
		t.Fatal(err)
	}
	limits, err := c.RateLimits(ctx)
	if err != nil || len(limits) != 2 || limits[0].Remaining != 4000 {
		t.Fatalf("limits = %+v, %v", limits, err)
	}

	joined := strings.Join(s.bodies, "\n")
	for _, want := range []string{`"draft":true`, `"state":"closed"`, `"team_reviewers":["platform"]`, `"reviewers":["octocat"]`, `"labels":["fleet"]`,
		`{"merge_method":"squash","sha":"abc"}`, `{"expected_head_sha":"abc"}`} {
		if !strings.Contains(joined, want) {
			t.Errorf("request bodies missing %s", want)
		}
	}

	// Every write fails against a repo with no routes.
	ctx = context.Background()
	if _, err := c.CreatePR(ctx, "acme/none", NewPR{}); err == nil {
		t.Fatal("create should fail")
	}
	if err := c.UpdatePR(ctx, "acme/none", 1, PRUpdate{Title: &title}); err == nil {
		t.Fatal("update should fail")
	}
	if err := c.AddLabels(ctx, "acme/none", 1, []string{"x"}); err == nil {
		t.Fatal("labels should fail")
	}
	if err := c.RequestReviewers(ctx, "acme/none", 1, []string{"x"}); err == nil {
		t.Fatal("reviewers should fail")
	}
	if err := c.Comment(ctx, "acme/none", 1, "x"); err == nil {
		t.Fatal("comment should fail")
	}
	if err := c.MergePR(ctx, "acme/none", 1, MergeMethodMerge, ""); err == nil || !strings.Contains(err.Error(), "merge acme/none#1") {
		t.Fatalf("merge should fail: %v", err)
	}
	if err := c.UpdateBranch(ctx, "acme/none", 1, ""); err == nil {
		t.Fatal("update branch should fail")
	}
	s.rest["DELETE /repos/acme/none/git/refs/heads/x"] = func(*http.Request, []byte) (any, int) { return map[string]any{"message": "denied"}, 403 }
	if err := c.DeleteBranch(ctx, "acme/none", "x"); err == nil {
		t.Fatal("delete should fail")
	}
	delete(s.rest, "GET /rate_limit")
	if _, err := c.RateLimits(ctx); err == nil {
		t.Fatal("rate limit should fail")
	}
}

func TestEnableAutoMerge(t *testing.T) {
	t.Parallel()
	var got map[string]any
	var query string
	s := &apiServer{gql: func(q string, vars map[string]any) (any, int) {
		query, got = q, vars
		if vars["id"] == "bad" {
			return nil, 400
		}
		return map[string]any{"enablePullRequestAutoMerge": map[string]any{"clientMutationId": nil}}, 200
	}}
	c := newClient(t, s)
	ctx := context.Background()
	if err := c.EnableAutoMerge(ctx, PR{Repo: "acme/api", Number: 5, NodeID: "PR_kw"}, MergeMethodSquash); err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(query, "mutation") || got["id"] != "PR_kw" || got["method"] != "SQUASH" {
		t.Fatalf("query %q vars %v", query, got)
	}
	if err := c.EnableAutoMerge(ctx, PR{Repo: "acme/api", Number: 5}, MergeMethodSquash); err == nil || !strings.Contains(err.Error(), "no node ID") {
		t.Fatalf("missing ID: %v", err)
	}
	if err := c.EnableAutoMerge(ctx, PR{Repo: "acme/api", Number: 5, NodeID: "bad"}, MergeMethodMerge); err == nil {
		t.Fatal("server error swallowed")
	}
}

func TestHelpersAndErrors(t *testing.T) {
	if _, _, err := SplitRepo("a/b/c"); err == nil {
		t.Fatal("SplitRepo accepted a/b/c")
	}
	users, teams := SplitReviewers([]string{"a", "org/t"})
	if len(users) != 1 || teams[0] != "t" {
		t.Fatal("SplitReviewers")
	}
	if repoFromAPIURL("x") != "x" || gqlString(`a"b`) != `"a\"b"` {
		t.Fatal("helpers")
	}
	if len(nonNil(nil)) != 0 {
		t.Fatal("nonNil")
	}
	// No token anywhere: construction fails like gh would.
	t.Setenv("GH_TOKEN", "")
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	if _, err := New(Options{Host: "github.com"}); err == nil {
		t.Fatal("expected missing token error")
	}
}

func TestCustomProperties(t *testing.T) {
	t.Parallel()
	pages := 0
	s := &apiServer{rest: map[string]func(*http.Request, []byte) (any, int){
		"GET /orgs/acme/properties/values": func(r *http.Request, _ []byte) (any, int) {
			pages++
			if r.URL.Query().Get("page") == "2" {
				return []any{map[string]any{"repository_full_name": "acme/last", "properties": []any{map[string]any{"property_name": "tier", "value": "2"}}}}, 200
			}
			out := []any{}
			for i := range 100 {
				props := []any{
					map[string]any{"property_name": "tier", "value": "1"},
					map[string]any{"property_name": "langs", "value": []any{"go", "rust"}},
					map[string]any{"property_name": "unset", "value": nil},
					map[string]any{"property_name": "flag", "value": true},
				}
				if i == 1 {
					props = []any{}
				}
				out = append(out, map[string]any{"repository_full_name": fmt.Sprintf("acme/r%d", i), "properties": props})
			}
			return out, 200
		},
		"GET /orgs/octocat/properties/values": func(*http.Request, []byte) (any, int) { return map[string]any{"message": "Not Found"}, 404 },
		"GET /orgs/denied/properties/values":  func(*http.Request, []byte) (any, int) { return map[string]any{"message": "nope"}, 403 },
		"GET /repos/acme/api/properties/values": func(*http.Request, []byte) (any, int) {
			return []any{map[string]any{"property_name": "team", "value": "payments"}}, 200
		},
		"GET /repos/acme/broken/properties/values": func(*http.Request, []byte) (any, int) { return map[string]any{"message": "boom"}, 400 },
	}}
	c := newClient(t, s)
	ctx := context.Background()
	props, err := c.OwnerProperties(ctx, "acme")
	if err != nil {
		t.Fatal(err)
	}
	r0 := props["acme/r0"]
	if pages != 2 || len(props) != 100 || r0["tier"] != "1" || r0["langs"] != "go,rust" || r0["flag"] != "true" || props["acme/last"]["tier"] != "2" {
		t.Fatalf("pages %d, props = %v", pages, r0)
	}
	if _, ok := r0["unset"]; ok {
		t.Fatal("unset value kept")
	}
	if _, ok := props["acme/r1"]; ok {
		t.Fatal("repo without properties listed")
	}
	if m, err := c.OwnerProperties(ctx, "octocat"); err != nil || len(m) != 0 {
		t.Fatalf("user account: %v %v", m, err)
	}
	if _, err := c.OwnerProperties(ctx, "denied"); err == nil {
		t.Fatal("403 hidden")
	}
	if m, err := c.RepoProperties(ctx, "acme/api"); err != nil || m["team"] != "payments" {
		t.Fatalf("repo props = %v, %v", m, err)
	}
	if m, err := c.RepoProperties(ctx, "acme/none"); err != nil || len(m) != 0 {
		t.Fatalf("404 repo props = %v, %v", m, err)
	}
	if _, err := c.RepoProperties(ctx, "acme/broken"); err == nil {
		t.Fatal("error hidden")
	}
	if _, err := c.RepoProperties(ctx, "bad"); err == nil {
		t.Fatal("invalid name accepted")
	}
}

func TestNewWithoutAuth(t *testing.T) {
	for _, k := range []string{"GH_TOKEN", "GITHUB_TOKEN", "GH_ENTERPRISE_TOKEN", "GITHUB_ENTERPRISE_TOKEN", "GH_HOST"} {
		t.Setenv(k, "")
	}
	t.Setenv("GH_CONFIG_DIR", t.TempDir())
	t.Setenv("PATH", "")
	if _, err := New(Options{}); !errors.Is(err, ErrAuthRequired) {
		t.Fatalf("err = %v", err)
	}
}
