package ghapi

import (
	"compress/gzip"
	"context"
	"io"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

// githubSchema loads GitHub's public GraphQL schema, from
// https://docs.github.com/public/fpt/schema.docs.graphql. Refresh it with
// `make schema`.
func githubSchema(t *testing.T) *ast.Schema {
	t.Helper()
	f, err := os.Open("testdata/schema.docs.graphql.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()
	zr, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	b, err := io.ReadAll(zr)
	if err != nil {
		t.Fatal(err)
	}
	s, err := gqlparser.LoadSchema(&ast.Source{Name: "github", Input: string(b)})
	if err != nil {
		t.Fatalf("load schema: %v", err)
	}
	return s
}

// TestQueriesMatchSchema sends every GraphQL query and mutation the client
// builds to a server that records them, then validates each against
// GitHub's schema, so a misspelled field or a wrong argument type fails
// here rather than against GitHub.
func TestQueriesMatchSchema(t *testing.T) {
	t.Parallel()
	schema := githubSchema(t)
	var mu sync.Mutex
	queries := map[string]string{}
	var current string
	s := &apiServer{gql: func(q string, _ map[string]any) (any, int) {
		mu.Lock()
		queries[current] = q
		mu.Unlock()
		return map[string]any{}, 200
	}}
	c := newClient(t, s)
	ctx := context.Background()
	repos := []Repo{{Owner: "acme", Name: "api"}, {Owner: "acme", Name: "web"}}
	calls := map[string]func(){
		"ListOwnerRepos":  func() { _, _ = c.ListOwnerRepos(ctx, "acme") },
		"GetRepo":         func() { _, _ = c.GetRepo(ctx, "acme/api") },
		"PathsExist":      func() { _, _ = c.PathsExist(ctx, repos, []string{"go.mod", ".github/workflows"}) },
		"PRsByHead":       func() { _, _ = c.PRsByHead(ctx, []string{"acme/api", "acme/web"}, "fleet/x") },
		"EnableAutoMerge": func() { _ = c.EnableAutoMerge(ctx, PR{Repo: "acme/api", Number: 1, NodeID: "PR_x"}, MergeMethodSquash) },
	}
	for name, call := range calls {
		mu.Lock()
		current = name
		mu.Unlock()
		call()
	}
	for name := range calls {
		q, ok := queries[name]
		if !ok {
			t.Errorf("%s sent no GraphQL request", name)
			continue
		}
		if _, errs := gqlparser.LoadQueryWithRules(schema, q, nil); errs != nil {
			t.Errorf("%s: %v\n%s", name, errs, q)
		}
	}
	// The check itself catches a bad field.
	if _, errs := gqlparser.LoadQueryWithRules(schema, `query { repository(owner: "a", name: "b") { mergeStatus } }`, nil); errs == nil || !strings.Contains(errs.Error(), "mergeStatus") {
		t.Errorf("a bad field passed: %v", errs)
	}
}
