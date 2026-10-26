package ghapi

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/cli/go-gh/v2/pkg/api"
	"golang.org/x/sync/errgroup"
)

// batchConcurrency is how many batched read queries run at once.
const batchConcurrency = 4

// forBatches calls run for each BatchSize slice [lo, hi) of n items,
// batchConcurrency at a time. A batch GitHub times out on is split in two
// and each half run again, down to single items, since a smaller query is
// what gets past a timeout.
func forBatches(ctx context.Context, n int, run func(ctx context.Context, lo, hi int) error) error {
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(batchConcurrency)
	for lo := 0; lo < n; lo += BatchSize {
		hi := min(lo+BatchSize, n)
		g.Go(func() error { return splitOnTimeout(gctx, lo, hi, run) })
	}
	return g.Wait()
}

// splitOnTimeout runs [lo, hi), halving it while GitHub times out.
func splitOnTimeout(ctx context.Context, lo, hi int, run func(ctx context.Context, lo, hi int) error) error {
	err := run(ctx, lo, hi)
	if err == nil || hi-lo <= 1 || !isTimeout(err) {
		return err
	}
	mid := lo + (hi-lo)/2
	if err := splitOnTimeout(ctx, lo, mid, run); err != nil {
		return err
	}
	return splitOnTimeout(ctx, mid, hi, run)
}

// isTimeout reports whether err is GitHub giving up on a query that took
// too long: a 502 or 504, or a GraphQL error saying so.
func isTimeout(err error) bool {
	var he *api.HTTPError
	if errors.As(err, &he) {
		return he.StatusCode == http.StatusBadGateway || he.StatusCode == http.StatusGatewayTimeout
	}
	var ge *api.GraphQLError
	if errors.As(err, &ge) {
		for _, e := range ge.Errors {
			msg := strings.ToLower(e.Message)
			if strings.Contains(msg, "timeout") || strings.Contains(msg, "timed out") || strings.Contains(msg, "timedout") {
				return true
			}
		}
	}
	return false
}
