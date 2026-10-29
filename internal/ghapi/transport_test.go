package ghapi

import (
	"context"
	"errors"
	"fmt"
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

	"github.com/rossedman/gh-fleet/internal/timing"
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
	var waits atomic.Int64
	tr.OnWait = func(why string, d time.Duration) {
		if why != "secondary rate limit" || d != 30*time.Second {
			t.Errorf("OnWait(%q, %s)", why, d)
		}
		waits.Add(1)
	}
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
	if tr.Pauses() != 1 || waits.Load() != 1 {
		t.Fatalf("pauses = %d, waits reported = %d", tr.Pauses(), waits.Load())
	}
	if calls.Load() != workers+1 {
		t.Fatalf("calls = %d, want %d", calls.Load(), workers+1)
	}
	if clock.Now().Sub(start) < 30*time.Second {
		t.Fatalf("clock advanced only %s", clock.Now().Sub(start))
	}
}

func TestClassify(t *testing.T) {
	t.Parallel()
	now := time.Unix(1_700_000_000, 0)
	reset := strconv.FormatInt(now.Add(90*time.Second).Unix(), 10)
	for _, tt := range []struct {
		name     string
		code     int
		resource string
		headers  map[string]string
		body     string
		kind     limitKind
		wait     time.Duration
	}{
		{"retry-after seconds", 403, "core", map[string]string{"Retry-After": "5"}, "", secondary, 5 * time.Second},
		{"retry-after date", 429, "core", map[string]string{"Retry-After": now.Add(10 * time.Second).UTC().Format(http.TimeFormat)}, "", secondary, 10 * time.Second},
		{"retry-after junk", 429, "core", map[string]string{"Retry-After": "soon"}, "", secondary, defaultRetryAfter},
		{"spent quota", 403, "core", map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Limit": "5000", "X-RateLimit-Reset": reset}, `{"message":"API rate limit exceeded"}`, spentQuota, 90 * time.Second},
		{"spent quota, no reset", 429, "search", map[string]string{"X-RateLimit-Remaining": "0"}, "", spentQuota, defaultRetryAfter},
		{"secondary, no wait given", 403, "core", nil, `{"message":"You have exceeded a secondary rate limit."}`, secondary, -1},
		{"abuse", 403, "core", nil, `{"message":"You have triggered an abuse detection mechanism."}`, secondary, -1},
		{"429, no hints", 429, "core", nil, "", secondary, -1},
		{"plain 403", 403, "core", nil, `{"message":"Resource not accessible by integration"}`, notLimited, 0},
		{"graphql rate limited", 200, "graphql", map[string]string{"X-RateLimit-Remaining": "0", "X-RateLimit-Reset": reset}, `{"errors":[{"type":"RATE_LIMITED"}]}`, spentQuota, 90 * time.Second},
		{"graphql last point", 200, "graphql", map[string]string{"X-RateLimit-Remaining": "0"}, `{"data":{}}`, notLimited, 0},
		{"ok", 200, "core", map[string]string{"X-RateLimit-Remaining": "0"}, "{}", notLimited, 0},
		{"server error", 500, "core", map[string]string{"Retry-After": "5"}, "", notLimited, 0},
	} {
		resp := &http.Response{StatusCode: tt.code, Header: http.Header{}, Body: io.NopCloser(strings.NewReader(tt.body))}
		for k, v := range tt.headers {
			resp.Header.Set(k, v)
		}
		got, err := classify(resp, tt.resource, now)
		if err != nil || got.kind != tt.kind || got.wait != tt.wait {
			t.Errorf("%s: classify = %+v, %v; want kind %d wait %s", tt.name, got, err, tt.kind, tt.wait)
			continue
		}
		// The body is still there for the caller.
		if b, _ := io.ReadAll(resp.Body); string(b) != tt.body {
			t.Errorf("%s: body = %q", tt.name, b)
		}
		if tt.kind == spentQuota && (got.quota == nil || !errors.Is(got.quota, ErrQuotaLow) || got.quota.Resource != tt.resource) {
			t.Errorf("%s: quota = %+v", tt.name, got.quota)
		}
	}
	// An unreadable body is an error.
	resp := &http.Response{StatusCode: 403, Header: http.Header{}, Body: io.NopCloser(errReader{})}
	if _, err := classify(resp, "core", now); err == nil {
		t.Error("unreadable body ignored")
	}
}

func TestSecondaryLimitWithoutWaitDoubles(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	limitOn := map[int64]bool{1: true, 2: true, 4: true}
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if limitOn[calls.Add(1)] {
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"message":"You have exceeded a secondary rate limit."}`)
			return
		}
		_, _ = io.WriteString(w, "{}")
	})
	tr, clock := newTestTransport(h)
	get(t, tr, "/repos/a/b")
	// A success resets the doubling.
	get(t, tr, "/repos/a/b")
	want := []time.Duration{time.Minute, 2 * time.Minute, time.Minute}
	if fmt.Sprint(clock.slept) != fmt.Sprint(want) {
		t.Fatalf("waits = %v, want %v", clock.slept, want)
	}

	// On the last attempt the response is returned.
	always := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTooManyRequests) })
	tr, clock = newTestTransport(always)
	if resp := get(t, tr, "/x"); resp.StatusCode != http.StatusTooManyRequests {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute}; fmt.Sprint(clock.slept) != fmt.Sprint(want) {
		t.Fatalf("waits = %v, want %v", clock.slept, want)
	}
}

func TestSpentQuotaStopsOrWaits(t *testing.T) {
	t.Parallel()
	start := time.Unix(1_700_000_000, 0)
	spent := func(resource string, status int, body string) (http.Handler, *atomic.Int64) {
		var calls atomic.Int64
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			if calls.Add(1) == 1 {
				w.Header().Set("X-RateLimit-Limit", "5000")
				w.Header().Set("X-RateLimit-Remaining", "0")
				w.Header().Set("X-RateLimit-Reset", strconv.FormatInt(start.Add(20*time.Second).Unix(), 10))
				w.Header().Set("X-RateLimit-Resource", resource)
				w.WriteHeader(status)
				_, _ = io.WriteString(w, body)
				return
			}
			_, _ = io.WriteString(w, "{}")
		}), &calls
	}

	// Core and GraphQL quota reset hourly: stop, without retrying.
	for _, tc := range []struct{ path, resource string }{{"/repos/a/b", "core"}, {"/graphql", "graphql"}} {
		status := http.StatusForbidden
		if tc.resource == "graphql" {
			status = http.StatusOK
		}
		h, calls := spent(tc.resource, status, `{"errors":[{"type":"RATE_LIMITED"}]}`)
		tr, _ := newTestTransport(h)
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com"+tc.path, strings.NewReader(`{"query":"query { x }"}`))
		if tc.resource == "core" {
			req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com"+tc.path, nil)
		}
		_, err := tr.RoundTrip(req)
		var qe *QuotaError
		if !errors.As(err, &qe) || qe.Resource != tc.resource || calls.Load() != 1 {
			t.Errorf("%s: err = %v after %d calls", tc.resource, err, calls.Load())
		}
	}

	// Search quota resets within a minute: wait, then retry.
	h, calls := spent("search", http.StatusForbidden, "")
	tr, clock := newTestTransport(h)
	var told []string
	tr.OnWait = func(why string, d time.Duration) { told = append(told, fmt.Sprintf("%s %s", why, d)) }
	if resp := get(t, tr, "/search/issues?q=x"); resp.StatusCode != 200 || calls.Load() != 2 {
		t.Fatalf("status %d after %d calls", resp.StatusCode, calls.Load())
	}
	if len(clock.slept) != 1 || clock.slept[0] != 20*time.Second || strings.Join(told, ",") != "search rate limit 20s" {
		t.Fatalf("slept %v, told %v", clock.slept, told)
	}
	// Still spent on the last attempt: stop.
	tr, _ = newTestTransport(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("X-RateLimit-Remaining", "0")
		w.WriteHeader(http.StatusForbidden)
	}))
	tr.MaxAttempts = 1
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com/search/code?q=x", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, ErrQuotaLow) {
		t.Fatalf("err = %v", err)
	}
	// A canceled wait stops.
	h, _ = spent("search", http.StatusForbidden, "")
	tr, _ = newTestTransport(h)
	tr.Sleep = func(context.Context, time.Duration) error { return context.Canceled }
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, "https://api.github.com/search/issues?q=x", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
	}
}

func TestContentCreationWindows(t *testing.T) {
	t.Parallel()
	ok := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = io.WriteString(w, "{}") })
	tr, clock := newTestTransport(ok)
	start := clock.Now()
	var told []string
	tr.OnWait = func(why string, d time.Duration) { told = append(told, fmt.Sprintf("%s %s", why, d)) }
	post := func() {
		t.Helper()
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com/repos/a/b/pulls", strings.NewReader("{}"))
		if _, err := tr.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
	}
	// 80 writes fit in the first minute; the 81st waits for it to pass.
	for range ContentPerMinute {
		post()
	}
	if !clock.Now().Equal(start) {
		t.Fatalf("waited %s within the minute", clock.Now().Sub(start))
	}
	post()
	if got := clock.Now().Sub(start); got != time.Minute {
		t.Fatalf("81st write after %s, want 1m0s", got)
	}
	// 500 fit in the hour; the 501st waits for the first to age out.
	for range ContentPerHour - ContentPerMinute - 1 {
		post()
	}
	before := clock.Now()
	post()
	if got := clock.Now().Sub(start); got != time.Hour || clock.Now().Equal(before) {
		t.Fatalf("501st write after %s, want 1h0m0s", got)
	}
	if len(told) == 0 || !strings.HasPrefix(told[0], "content creation limit ") {
		t.Fatalf("OnWait = %v", told)
	}
	// Reads, and writes that create nothing, don't count.
	n := len(clock.slept)
	get(t, tr, "/repos/a/b")
	for _, path := range []string{"/repos/a/b/issues/1/labels", "/repos/a/b/pulls/1/requested_reviewers"} {
		req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com"+path, strings.NewReader("{}"))
		if _, err := tr.RoundTrip(req); err != nil {
			t.Fatal(err)
		}
	}
	if len(clock.slept) != n {
		t.Fatal("a request that creates nothing waited for the content limit")
	}
	// A canceled wait stops.
	tr.Sleep = func(context.Context, time.Duration) error { return context.Canceled }
	for range ContentPerMinute {
		tr.content[0].add(clock.Now())
	}
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodPost, "https://api.github.com/repos/a/b/pulls", strings.NewReader("{}"))
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) {
		t.Fatalf("err = %v", err)
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
	var told []string
	tr.OnWait = func(why string, d time.Duration) { told = append(told, fmt.Sprintf("%s %s", why, d)) }
	get(t, tr, "/search/issues?q=x")
	get(t, tr, "/search/issues?q=x")
	if len(clock.slept) != 1 || clock.slept[0] != 40*time.Second {
		t.Fatalf("slept %v", clock.slept)
	}
	if strings.Join(told, ",") != "search rate limit 40s" {
		t.Fatalf("OnWait = %v", told)
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
	for _, tt := range []struct {
		method, path string
		want         bool
	}{
		{"POST", "/repos/a/b/pulls", true},
		{"POST", "/repos/a/b/issues", true},
		{"POST", "/repos/a/b/issues/1/comments", true},
		{"POST", "/repos/a/b/pulls/1/reviews", true},
		{"POST", "/repos/a/b/issues/1/labels", false},
		{"POST", "/repos/a/b/pulls/1/requested_reviewers", false},
		{"PATCH", "/repos/a/b/pulls/1", false},
		{"PUT", "/repos/a/b/pulls/1/merge", false},
		{"DELETE", "/repos/a/b/git/refs/heads/x", false},
		{"POST", "/graphql", false},
		{"GET", "/repos/a/b/pulls", false},
	} {
		if got := createsContent(tt.method, tt.path); got != tt.want {
			t.Errorf("createsContent(%s %s) = %v", tt.method, tt.path, got)
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

// errTransport fails every request.
type errTransport struct{}

// RoundTrip returns an error.
func (errTransport) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("connection refused")
}

func TestTransportRecordsTimings(t *testing.T) {
	t.Parallel()
	var calls atomic.Int64
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		switch calls.Add(1) {
		case 1:
			w.WriteHeader(http.StatusForbidden)
			_, _ = io.WriteString(w, `{"message":"You have exceeded a secondary rate limit."}`)
		case 2:
			w.WriteHeader(http.StatusBadGateway)
		default:
			_, _ = io.WriteString(w, "{}")
		}
	})
	tr, _ := newTestTransport(h)
	tr.Timings = timing.New(nil)
	if resp := get(t, tr, "/repos/a/b"); resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	got := map[string]timing.Stat{}
	for _, s := range tr.Timings.Stats() {
		got[s.Name] = s
	}
	if s := got["api core"]; s.Count != 3 {
		t.Fatalf("api core = %+v", s)
	}
	// The secondary limit's minute is spent waiting before the next try.
	if s := got["api wait: rate limits"]; s.Count != 1 || s.Total != time.Minute {
		t.Fatalf("rate limit wait = %+v", s)
	}
	// The 502 came on the second attempt, which backs off 2s.
	if s := got["api wait: retry backoff"]; s.Count != 1 || s.Total != 2*time.Second {
		t.Fatalf("backoff = %+v", s)
	}

	failing := NewTransport(errTransport{}, 0)
	failing.Timings = timing.New(nil)
	req, _ := http.NewRequestWithContext(t.Context(), http.MethodPost, "https://api.github.com/graphql", strings.NewReader(`{"query":"query{viewer{login}}"}`))
	if _, err := failing.RoundTrip(req); err == nil {
		t.Fatal("no error")
	}
	if s := failing.Timings.Stats(); len(s) != 1 || s[0].Name != "api graphql" || s[0].Count != 1 {
		t.Fatalf("stats = %+v", s)
	}
}

func TestInFlightCap(t *testing.T) {
	t.Parallel()
	var sent atomic.Int64
	h := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		sent.Add(1)
		_, _ = io.WriteString(w, "{}")
	})
	tr, _ := newTestTransport(h)
	if cap(tr.inFlight) != DefaultMaxInFlight || DefaultMaxInFlight >= 100 {
		t.Fatalf("cap = %d", cap(tr.inFlight))
	}
	// Every slot taken: a request waits, and gives up with its context.
	for range DefaultMaxInFlight {
		tr.inFlight <- struct{}{}
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/x", nil)
	if _, err := tr.RoundTrip(req); !errors.Is(err, context.Canceled) || sent.Load() != 0 {
		t.Fatalf("full: %v, %d sent", err, sent.Load())
	}
	// A freed slot lets it through, and it gives the slot back.
	<-tr.inFlight
	if resp := get(t, tr, "/x"); resp.StatusCode != http.StatusOK || sent.Load() != 1 || len(tr.inFlight) != DefaultMaxInFlight-1 {
		t.Fatalf("freed: %d sent, %d slots taken", sent.Load(), len(tr.inFlight))
	}
	// A transport built without NewTransport has no cap.
	bare := &Transport{Base: handlerTransport{h}}
	if resp := get(t, bare, "/x"); resp.StatusCode != http.StatusOK {
		t.Fatalf("bare: %d", resp.StatusCode)
	}
}
