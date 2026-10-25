package pr

import (
	"context"
	"errors"
	"testing"

	"github.com/rossedman/gh-fleet/internal/ghapi"
	"github.com/rossedman/gh-fleet/internal/ghapi/fake"
)

const marker = "<!-- gh-fleet:campaign=x -->"

// desired returns a Desired PR for repo.
func desired(repo string) Desired {
	return Desired{Repo: repo, Title: "t", Body: "b\n" + marker, Head: "fleet/x", Base: "main", Labels: []string{"fleet", "sec"}, Reviewers: []string{"octocat"}}
}

func TestIndexCountsAndFallsBackToSearch(t *testing.T) {
	t.Parallel()
	gh := fake.New()
	gh.PRs = []*ghapi.PR{
		{Repo: "acme/a", Number: 1, State: ghapi.StateOpen, HeadRef: "fleet/x", Body: marker},
		{Repo: "acme/b", Number: 2, State: ghapi.StateMerged, HeadRef: "fleet/x", Body: marker},
		{Repo: "acme/c", Number: 3, State: ghapi.StateOpen, HeadRef: "renamed", Body: marker},
		{Repo: "other/d", Number: 4, State: ghapi.StateOpen, HeadRef: "renamed", Body: marker},
	}
	m := NewManager(gh, 0)
	if err := m.Index(context.Background(), []string{"acme/a", "acme/b", "acme/c", "acme/e"}, "fleet/x", marker, []string{"user:acme"}); err != nil {
		t.Fatal(err)
	}
	if m.OpenCount() != 2 || len(m.All()) != 3 || m.Existing("acme/c").Number != 3 || m.Existing("acme/e") != nil {
		t.Fatalf("index = %+v", m.All())
	}

	gh.Errors["SearchCampaignPRs"] = errors.New("search down")
	if err := m.Index(context.Background(), []string{"acme/e"}, "fleet/x", marker, nil); err == nil {
		t.Fatal("expected search error")
	}
	gh.Errors = map[string]error{"PRsByHead": errors.New("down")}
	if err := m.Index(context.Background(), []string{"acme/e"}, "fleet/x", marker, nil); err == nil {
		t.Fatal("expected lookup error")
	}
}

func TestSyncCreateUpdateReopen(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gh := fake.New()
	m := NewManager(gh, 0)
	p, action, err := m.Sync(ctx, desired("acme/a"), nil, false)
	if err != nil || action != Created || p.Number != 1 || gh.Reviewers["acme/a#1"][0] != "octocat" || len(gh.PRs[0].Labels) != 2 {
		t.Fatalf("create: %+v %s %v", p, action, err)
	}
	existing := m.Existing("acme/a")
	if _, action, _ := m.Sync(ctx, desired("acme/a"), existing, false); action != Unchanged {
		t.Fatalf("action = %s", action)
	}
	d := desired("acme/a")
	d.Title, d.Labels = "new", append(d.Labels, "extra")
	p, action, err = m.Sync(ctx, d, m.Existing("acme/a"), false)
	if err != nil || action != Updated || gh.PRs[0].Title != "new" || len(gh.PRs[0].Labels) != 3 {
		t.Fatalf("update: %+v %s %v", gh.PRs[0], action, err)
	}
	// Only a missing label changes.
	d.Labels = append(d.Labels, "more")
	if _, action, _ := m.Sync(ctx, d, m.Existing("acme/a"), false); action != Updated {
		t.Fatalf("label-only update = %s", action)
	}

	gh.SetPRState("acme/a", 1, ghapi.StateClosed, "", "")
	closed := *m.Existing("acme/a")
	closed.State = ghapi.StateClosed
	p, action, err = m.Sync(ctx, d, &closed, true)
	if err != nil || action != Reopened || p.State != ghapi.StateOpen || gh.PRs[0].State != ghapi.StateOpen {
		t.Fatalf("reopen: %+v %s %v", p, action, err)
	}

	// GitHub refuses to reopen: a new PR is opened instead.
	gh.Errors["UpdatePR"] = errors.New("cannot reopen")
	closed.State = ghapi.StateClosed
	p, action, err = m.Sync(ctx, d, &closed, true)
	if err != nil || action != Created || p.Number != 2 {
		t.Fatalf("fallback: %+v %s %v", p, action, err)
	}
	// Other update errors are returned.
	if _, _, err := m.Sync(ctx, desired("acme/a"), m.Existing("acme/a"), false); err == nil {
		t.Fatal("expected update error")
	}
}

func TestSyncErrors(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	for _, method := range []string{"CreatePR", "AddLabels", "RequestReviewers"} {
		gh := fake.New()
		gh.Errors[method] = errors.New("boom")
		if _, _, err := NewManager(gh, 0).Sync(ctx, desired("acme/a"), nil, false); err == nil {
			t.Errorf("%s error not returned", method)
		}
	}
	gh := fake.New()
	m := NewManager(gh, 0)
	if _, _, err := m.Sync(ctx, desired("acme/a"), nil, false); err != nil {
		t.Fatal(err)
	}
	gh.Errors["AddLabels"] = errors.New("boom")
	d := desired("acme/a")
	d.Labels = []string{"new"}
	if _, _, err := m.Sync(ctx, d, m.Existing("acme/a"), false); err == nil {
		t.Fatal("expected label error")
	}
}

func TestReserveAndClose(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	gh := fake.New()
	m := NewManager(gh, 2)
	for i, want := range []bool{true, true, false} {
		if got := m.Reserve(); got != want {
			t.Fatalf("Reserve #%d = %v", i+1, got)
		}
	}
	m.Release()
	if !m.Reserve() {
		t.Fatal("release did not free a slot")
	}

	m = NewManager(gh, 0)
	p, _, _ := m.Sync(ctx, desired("acme/a"), nil, false)
	m.open = 1
	if err := m.Close(ctx, p, "closing", true); err != nil {
		t.Fatal(err)
	}
	if m.OpenCount() != 0 || m.Existing("acme/a").State != ghapi.StateClosed || len(gh.DeletedBranches) != 1 || len(gh.Comments["acme/a#1"]) != 1 {
		t.Fatalf("close: %d %v", m.OpenCount(), gh.DeletedBranches)
	}
	for _, method := range []string{"Comment", "UpdatePR", "DeleteBranch"} {
		gh := fake.New()
		gh.Errors[method] = errors.New("boom")
		if err := NewManager(gh, 0).Close(ctx, p, "c", true); err == nil {
			t.Errorf("%s error not returned", method)
		}
	}
}
