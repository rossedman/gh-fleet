package ghapi

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/cli/go-gh/v2/pkg/api"
)

func TestForBatchesSplitsOnTimeout(t *testing.T) {
	t.Parallel()
	// GitHub times out on any query over 10 items.
	var mu sync.Mutex
	var done []int
	var calls atomic.Int64
	err := forBatches(context.Background(), 120, func(_ context.Context, lo, hi int) error {
		calls.Add(1)
		if hi-lo > 10 {
			return &api.HTTPError{StatusCode: http.StatusBadGateway}
		}
		mu.Lock()
		defer mu.Unlock()
		for i := lo; i < hi; i++ {
			done = append(done, i)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(done)
	if len(done) != 120 || done[0] != 0 || done[119] != 119 {
		t.Fatalf("covered %d items", len(done))
	}
	// 50 -> 25 -> 12/13 -> 6/7 takes 1+2+4+8 calls per full batch, and
	// the last 20 -> 10 takes 1+2.
	if n := calls.Load(); n != 2*15+3 {
		t.Errorf("%d calls", n)
	}

	// Other errors, and a single item that times out, fail the run.
	boom := errors.New("boom")
	if err := forBatches(context.Background(), 5, func(context.Context, int, int) error { return boom }); !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
	timeout := &api.GraphQLError{Errors: []api.GraphQLErrorItem{{Message: "Something went wrong while executing your query. This may be the result of a timeout"}}}
	if err := forBatches(context.Background(), 2, func(context.Context, int, int) error { return timeout }); err == nil {
		t.Error("timeout on one item swallowed")
	}
	// A failing second half fails the run.
	err = forBatches(context.Background(), 4, func(_ context.Context, lo, hi int) error {
		if hi-lo > 1 {
			return timeout
		}
		if lo == 3 {
			return boom
		}
		return nil
	})
	if !errors.Is(err, boom) {
		t.Errorf("err = %v", err)
	}
}

func TestIsTimeout(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		err  error
		want bool
	}{
		{&api.HTTPError{StatusCode: 502}, true},
		{fmt.Errorf("wrapped: %w", &api.HTTPError{StatusCode: 504}), true},
		{&api.HTTPError{StatusCode: 500}, false},
		{&api.GraphQLError{Errors: []api.GraphQLErrorItem{{Message: "query timed out"}}}, true},
		{&api.GraphQLError{Errors: []api.GraphQLErrorItem{{Type: "TIMEDOUT", Message: "timedout"}}}, true},
		{&api.GraphQLError{Errors: []api.GraphQLErrorItem{{Type: "NOT_FOUND", Message: "Could not resolve"}}}, false},
		{errors.New("x"), false},
	} {
		if got := isTimeout(tc.err); got != tc.want {
			t.Errorf("isTimeout(%v) = %v", tc.err, got)
		}
	}
}

func TestGraphQLTimeoutRetriedOnce(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusBadGateway)
	})
	tr, _ := newTestTransport(h)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com/graphql", strings.NewReader(`{"query":"query { x }"}`))
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode != http.StatusBadGateway || calls.Load() != 2 {
		t.Fatalf("status %d after %d calls, want 2", resp.StatusCode, calls.Load())
	}
	// REST keeps every attempt.
	calls.Store(0)
	get(t, tr, "/repos/a/b")
	if calls.Load() != DefaultMaxAttempts {
		t.Fatalf("REST: %d calls", calls.Load())
	}
}

func TestBatchedLookupErrors(t *testing.T) {
	t.Parallel()
	s := &apiServer{gql: func(string, map[string]any) (any, int) { return nil, 400 }}
	c := newClient(t, s)
	ctx := context.Background()
	if _, err := c.PRsByHead(ctx, []string{"acme/api"}, "fleet/x"); err == nil {
		t.Error("PRsByHead error swallowed")
	}
	if _, err := c.PRsByHead(ctx, []string{"bad"}, "fleet/x"); err == nil {
		t.Error("PRsByHead accepted a bad name")
	}
	if _, err := c.GetRepos(ctx, []string{"acme/api"}); err == nil {
		t.Error("GetRepos error swallowed")
	}
}
