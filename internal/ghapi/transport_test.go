package ghapi

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"
)

// handlerTransport serves requests from an http.Handler in process.
type handlerTransport struct{ h http.Handler }

// RoundTrip records the handler's response.
func (t handlerTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	rec := httptest.NewRecorder()
	t.h.ServeHTTP(rec, r)
	resp := rec.Result()
	resp.Request = r
	return resp, nil
}

// fakeClock is a manually advanced clock whose sleeps advance it.
type fakeClock struct {
	mu    sync.Mutex
	now   time.Time
	slept []time.Duration
}

// Now returns the current fake time.
func (c *fakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// Sleep advances the clock by d.
func (c *fakeClock) Sleep(ctx context.Context, d time.Duration) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	c.slept = append(c.slept, d)
	return nil
}

// newTestTransport returns a transport over h with a fake clock.
func newTestTransport(h http.Handler) (*Transport, *fakeClock) {
	clock := &fakeClock{now: time.Unix(1_700_000_000, 0)}
	t := NewTransport(handlerTransport{h}, 0)
	t.Now, t.Sleep = clock.Now, clock.Sleep
	t.Jitter = func(time.Duration) time.Duration { return 0 }
	return t, clock
}

// get sends a GET through t.
func get(t *testing.T, tr http.RoundTripper, path string) *http.Response {
	t.Helper()
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com"+path, nil)
	resp, err := tr.RoundTrip(req)
	if err != nil {
		t.Fatal(err)
	}
	return resp
}

func TestSecondaryLimitPausesAllWorkersAndResumes(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	var limited atomic.Bool
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if limited.CompareAndSwap(false, true) {
			w.Header().Set("Retry-After", "30")
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"message":"You have exceeded a secondary rate limit"}`)
			return
		}
		_, _ = io.WriteString(w, `{}`)
	})
	tr, clock := newTestTransport(h)
	start := clock.Now()

	const workers = 8
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for range workers {
		wg.Add(1)
		go func() {
			defer wg.Done()
			req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com/repos/a/b/pulls", strings.NewReader(`{"title":"x"}`))
			resp, err := tr.RoundTrip(req)
			if err != nil {
				errs <- err
				return
			}
			if resp.StatusCode != http.StatusOK {
				errs <- errors.New(resp.Status)
			}
			_ = resp.Body.Close()
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("request failed: %v", err)
	}
	if tr.Pauses() != 1 {
		t.Fatalf("pauses = %d", tr.Pauses())
	}
	if calls.Load() != workers+1 {
		t.Fatalf("calls = %d, want %d", calls.Load(), workers+1)
	}
	if clock.Now().Sub(start) < 30*time.Second {
		t.Fatalf("clock advanced only %s", clock.Now().Sub(start))
	}
}

func TestRetryAfterFormats(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	for _, tt := range []struct {
		code int
		ra   string
		want time.Duration
		ok   bool
	}{
		{403, "5", 5 * time.Second, true},
		{429, now.Add(10 * time.Second).UTC().Format(http.TimeFormat), 10 * time.Second, true},
		{429, "soon", defaultRetryAfter, true},
		{403, "", 0, false},
		{500, "5", 0, false},
	} {
		resp := &http.Response{StatusCode: tt.code, Header: http.Header{}}
		if tt.ra != "" {
			resp.Header.Set("Retry-After", tt.ra)
		}
		got, ok := secondaryLimit(resp, now)
		if ok != tt.ok || got != tt.want {
			t.Errorf("secondaryLimit(%d, %q) = %s, %v", tt.code, tt.ra, got, ok)
		}
	}
}

func TestServerErrorsBackOffThenGiveUp(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if calls.Add(1) < 3 {
			w.WriteHeader(http.StatusBadGateway)
			return
		}
		_, _ = io.WriteString(w, "ok")
	})
	tr, clock := newTestTransport(h)
	resp := get(t, tr, "/x")
	if resp.StatusCode != 200 || tr.Retries() != 2 {
		t.Fatalf("status %d after %d retries", resp.StatusCode, tr.Retries())
	}
	if len(clock.slept) != 2 || clock.slept[0] != time.Second || clock.slept[1] != 2*time.Second {
		t.Fatalf("backoff = %v", clock.slept)
	}

	always := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusServiceUnavailable) })
	tr, _ = newTestTransport(always)
	if resp := get(t, tr, "/x"); resp.StatusCode != http.StatusServiceUnavailable || tr.Retries() != DefaultMaxAttempts-1 {
		t.Fatalf("status %d, retries %d", resp.StatusCode, tr.Retries())
	}
}

// quotaHandler reports remaining quota for a resource.
func quotaHandler(resource string, remaining int, reset time.Time) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "5000")
		w.Header().Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
		w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(reset.Unix(), 10))
		w.Header().Set("X-RateLimit-Resource", resource)
		_, _ = io.WriteString(w, "{}")
	})
}

func TestQuotaHeadroomStopsRequests(t *testing.T) {
	t.Parallel()
	reset := time.Unix(1_700_000_000, 0).Add(time.Hour)
	tr, _ := newTestTransport(quotaHandler("core", 499, reset))
	get(t, tr, "/repos/a/b")
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com/repos/a/b", nil)
	_, err := tr.RoundTrip(req)
	var qe *QuotaError
	if !errors.As(err, &qe) || !errors.Is(err, ErrQuotaLow) || qe.Remaining != 499 {
		t.Fatalf("err = %v", err)
	}
	if !strings.Contains(qe.Error(), "core quota at 499 of 5000") {
		t.Fatal(qe.Error())
	}
	if q := tr.Quotas(); len(q) != 1 || q[0].Resource != "core" {
		t.Fatalf("quotas = %+v", q)
	}

	// Plenty left: requests go through.
	tr, _ = newTestTransport(quotaHandler("core", 4000, reset))
	get(t, tr, "/a")
	get(t, tr, "/a")
}

func TestSearchQuotaWaitsForReset(t *testing.T) {
	t.Parallel()
	start := time.Unix(1_700_000_000, 0)
	tr, clock := newTestTransport(quotaHandler("search", 0, start.Add(40*time.Second)))
	get(t, tr, "/search/issues?q=x")
	get(t, tr, "/search/issues?q=x")
	if len(clock.slept) != 1 || clock.slept[0] != 40*time.Second {
		t.Fatalf("slept %v", clock.slept)
	}
}

func TestWriteLimiterAndClassification(t *testing.T) {
	t.Parallel()
	for _, tt := range []struct {
		method, path, body string
		want               bool
	}{
		{"GET", "/repos/a/b", "", false},
		{"POST", "/repos/a/b/pulls", "{}", true},
		{"POST", "/graphql", `{"query":"query { viewer { login } }"}`, false},
		{"POST", "/graphql", `{"query":"mutation { x }"}`, true},
		{"POST", "/graphql", `{"query":"\n  mutation { x }"}`, true},
		{"POST", "/graphql", `{"variables":{}}`, false},
		{"DELETE", "/repos/a/b/git/refs/heads/x", "", true},
	} {
		if got := isWrite(tt.method, tt.path, []byte(tt.body)); got != tt.want {
			t.Errorf("isWrite(%s %s %s) = %v", tt.method, tt.path, tt.body, got)
		}
	}
	for path, want := range map[string]string{"/graphql": "graphql", "/search/code": "code_search", "/search/issues": "search", "/repos/a": "core"} {
		if got := resourceFor(path); got != want {
			t.Errorf("resourceFor(%s) = %s", path, got)
		}
	}

	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "{}") })
	tr, _ := newTestTransport(ok)
	tr.Writes = rate.NewLimiter(rate.Inf, 1)
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com/repos/a/b/pulls", strings.NewReader("{}"))
	if _, err := tr.RoundTrip(req); err != nil {
		t.Fatal(err)
	}
	// A canceled context stops a throttled write.
	tr.Writes = rate.NewLimiter(rate.Every(time.Hour), 1)
	tr.Writes.Allow()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	req, _ = http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/repos/a/b/pulls", strings.NewReader("{}"))
	if _, err := tr.RoundTrip(req); err == nil {
		t.Fatal("expected canceled write to fail")
	}
}

func TestTransportDefaults(t *testing.T) {
	t.Parallel()
	tr := &Transport{}
	if tr.base() != http.DefaultTransport || tr.jitter(0) != 0 || tr.jitter(time.Second) >= time.Second {
		t.Fatal("defaults")
	}
	if d := time.Since(tr.now()); d > time.Minute {
		t.Fatal("now")
	}
	if err := tr.sleep(context.Background(), time.Millisecond); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tr.sleep(ctx, time.Hour); err == nil {
		t.Fatal("sleep ignored cancellation")
	}
	tr.pause(time.Hour)
	if err := tr.waitForPause(ctx); err == nil {
		t.Fatal("waitForPause ignored cancellation")
	}
	tr.recordQuota(http.Header{"X-Ratelimit-Limit": {"x"}})
	if len(tr.Quotas()) != 0 {
		t.Fatal("bad headers recorded")
	}
	// A network error is returned as is.
	failing := &Transport{Base: roundTripFunc(func(*http.Request) (*http.Response, error) { return nil, errors.New("dial") })}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://x/", nil)
	if _, err := failing.RoundTrip(req); err == nil {
		t.Fatal("expected network error")
	}
	// An unreadable body fails early.
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodPost, "https://x/", io.NopCloser(errReader{}))
	if _, err := failing.RoundTrip(req); err == nil {
		t.Fatal("expected body error")
	}
}

// roundTripFunc adapts a function to http.RoundTripper.
type roundTripFunc func(*http.Request) (*http.Response, error)

// RoundTrip calls f.
func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// errReader always fails.
type errReader struct{}

// Read returns an error.
func (errReader) Read([]byte) (int, error) { return 0, errors.New("read failed") }
