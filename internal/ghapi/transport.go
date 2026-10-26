package ghapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"golang.org/x/time/rate"
)

// ErrQuotaLow is returned instead of sending a request that would eat into
// the reserved headroom of the primary rate limit.
var ErrQuotaLow = errors.New("primary rate limit headroom reached")

// QuotaError says which quota ran low and when it resets.
type QuotaError struct {
	Resource  string
	Remaining int
	Limit     int
	Reset     time.Time
}

// Error describes the quota and its reset time.
func (e *QuotaError) Error() string {
	return fmt.Sprintf("%s: %s quota at %d of %d, resets %s", ErrQuotaLow, e.Resource, e.Remaining, e.Limit, e.Reset.Format(time.Kitchen))
}

// Unwrap makes errors.Is(err, ErrQuotaLow) work.
func (e *QuotaError) Unwrap() error { return ErrQuotaLow }

// Defaults for Transport.
const (
	DefaultWriteRate   = 1.0 // writes per second
	DefaultMaxAttempts = 5
	DefaultHeadroom    = 0.10
	defaultRetryAfter  = 60 * time.Second
	baseBackoff        = time.Second
	// maxSecondaryWait caps the doubling wait for repeated secondary
	// rate limits that don't say how long to wait.
	maxSecondaryWait = 15 * time.Minute
)

// GitHub's secondary limits on content-creating requests.
const (
	ContentPerMinute = 80
	ContentPerHour   = 500
)

// Transport is an http.RoundTripper shared by every worker. It follows
// GitHub's rate limit rules: writes are spaced out and kept within the
// content creation limits, a secondary rate limit pauses every request for
// as long as GitHub asks (or a minute, doubling, when it doesn't say),
// spent primary quota stops requests until it resets, and requests are
// refused once primary quota drops into the reserved headroom. Server
// errors are retried with exponential backoff and jitter.
type Transport struct {
	// Base sends the request. Defaults to http.DefaultTransport.
	Base http.RoundTripper
	// Writes throttles content-creating requests. Nil means no limit.
	Writes *rate.Limiter
	// MaxAttempts caps attempts per request. Defaults to 5.
	MaxAttempts int
	// Headroom is the fraction of primary quota kept in reserve.
	Headroom float64
	// Sleep waits for d or until ctx is done. Defaults to a timer.
	Sleep func(ctx context.Context, d time.Duration) error
	// Now returns the current time. Defaults to time.Now.
	Now func() time.Time
	// Jitter returns a random duration in [0, d). Defaults to math/rand.
	Jitter func(d time.Duration) time.Duration
	// OnWait, when set, is told each time every request starts waiting on
	// a rate limit, and for how long, so a long silence can be explained.
	OnWait func(why string, d time.Duration)

	mu          sync.Mutex
	pausedUntil time.Time
	quotas      map[string]RateLimit
	pauses      atomic.Int64
	retries     atomic.Int64
	// secondaries counts secondary rate limits in a row, for the doubling
	// wait when GitHub doesn't say how long to wait.
	secondaries atomic.Int64

	// content holds the content creation windows; nil means no limit.
	content      []*window
	contentMu    sync.Mutex
	contentNoted time.Time
}

// NewTransport returns a Transport with writes limited to writesPerSecond.
func NewTransport(base http.RoundTripper, writesPerSecond float64) *Transport {
	t := &Transport{Base: base, MaxAttempts: DefaultMaxAttempts, Headroom: DefaultHeadroom}
	if writesPerSecond > 0 {
		t.Writes = rate.NewLimiter(rate.Limit(writesPerSecond), 1)
	}
	t.content = []*window{{n: ContentPerMinute, period: time.Minute}, {n: ContentPerHour, period: time.Hour}}
	return t
}

// Pauses returns how many times a secondary rate limit paused all workers.
func (t *Transport) Pauses() int { return int(t.pauses.Load()) }

// Retries returns how many server errors were retried.
func (t *Transport) Retries() int { return int(t.retries.Load()) }

// Quotas returns the last primary quota seen per resource.
func (t *Transport) Quotas() []RateLimit {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]RateLimit, 0, len(t.quotas))
	for _, q := range t.quotas {
		out = append(out, q)
	}
	return out
}

// RoundTrip sends req, applying the rate limiting and retry policy.
func (t *Transport) RoundTrip(req *http.Request) (*http.Response, error) {
	ctx := req.Context()
	var body []byte
	if req.Body != nil {
		b, err := io.ReadAll(req.Body)
		_ = req.Body.Close()
		if err != nil {
			return nil, err
		}
		body = b
	}
	write := isWrite(req.Method, req.URL.Path, body)
	content := createsContent(req.Method, req.URL.Path)
	resource := resourceFor(req.URL.Path)
	attempts := t.MaxAttempts
	if attempts < 1 {
		attempts = DefaultMaxAttempts
	}

	for attempt := 1; ; attempt++ {
		if err := t.waitForPause(ctx); err != nil {
			return nil, err
		}
		if err := t.checkQuota(ctx, resource); err != nil {
			return nil, err
		}
		if content {
			if err := t.waitForContent(ctx); err != nil {
				return nil, err
			}
		}
		if write {
			if t.Writes != nil {
				if err := t.Writes.Wait(ctx); err != nil {
					return nil, err
				}
			}
		}
		r := req.Clone(ctx)
		if body != nil {
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.GetBody = func() (io.ReadCloser, error) { return io.NopCloser(bytes.NewReader(body)), nil }
			r.ContentLength = int64(len(body))
		}
		resp, err := t.base().RoundTrip(r)
		if err != nil {
			return nil, err
		}
		t.recordQuota(resp.Header)
		last := attempt >= attempts

		limit, err := classify(resp, resource, t.now())
		if err != nil {
			return nil, err
		}
		switch limit.kind {
		case spentQuota:
			if !strings.HasSuffix(resource, "search") || last {
				drain(resp)
				return nil, limit.quota
			}
			// Search quotas reset within a minute: wait it out.
			drain(resp)
			if t.OnWait != nil {
				t.OnWait(resource+" rate limit", limit.wait)
			}
			if err := t.sleep(ctx, limit.wait); err != nil {
				return nil, err
			}
			continue
		case secondary:
			if last {
				return resp, nil
			}
			drain(resp)
			wait := limit.wait
			if wait < 0 {
				// GitHub didn't say how long: at least a minute, doubling
				// while the limit keeps hitting.
				n := min(t.secondaries.Add(1)-1, 10)
				wait = min(defaultRetryAfter<<n, maxSecondaryWait)
			}
			t.pause(wait)
			continue
		}
		t.secondaries.Store(0)
		if isRetryableStatus(resp.StatusCode) && !last {
			drain(resp)
			t.retries.Add(1)
			d := baseBackoff << (attempt - 1)
			d += t.jitter(d)
			if err := t.sleep(ctx, d); err != nil {
				return nil, err
			}
			continue
		}
		return resp, nil
	}
}

// waitForContent waits until a content-creating request fits in every
// content creation window, then counts it.
func (t *Transport) waitForContent(ctx context.Context) error {
	for {
		t.contentMu.Lock()
		now := t.now()
		var d time.Duration
		for _, w := range t.content {
			d = max(d, w.wait(now))
		}
		if d <= 0 {
			for _, w := range t.content {
				w.add(now)
			}
			t.contentMu.Unlock()
			return nil
		}
		// Every worker waits at once; one note says so.
		note := now.Add(d).After(t.contentNoted)
		if note {
			t.contentNoted = now.Add(d)
		}
		t.contentMu.Unlock()
		if note && t.OnWait != nil {
			t.OnWait("content creation limit", d)
		}
		if err := t.sleep(ctx, d); err != nil {
			return err
		}
	}
}

// window counts events in a rolling period, as GitHub counts its content
// creation limits.
type window struct {
	n      int
	period time.Duration
	times  []time.Time
}

// wait returns how long until another event fits in the window.
func (w *window) wait(now time.Time) time.Duration {
	cutoff := now.Add(-w.period)
	i := 0
	for i < len(w.times) && !w.times[i].After(cutoff) {
		i++
	}
	w.times = w.times[i:]
	if len(w.times) < w.n {
		return 0
	}
	return w.times[len(w.times)-w.n].Add(w.period).Sub(now)
}

// add counts an event at now.
func (w *window) add(now time.Time) {
	w.times = append(w.times, now)
}

// base returns the underlying RoundTripper.
func (t *Transport) base() http.RoundTripper {
	if t.Base != nil {
		return t.Base
	}
	return http.DefaultTransport
}

// now returns the current time from the injected clock.
func (t *Transport) now() time.Time {
	if t.Now != nil {
		return t.Now()
	}
	return time.Now()
}

// sleep waits using the injected sleeper.
func (t *Transport) sleep(ctx context.Context, d time.Duration) error {
	if t.Sleep != nil {
		return t.Sleep(ctx, d)
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

// jitter returns a random extra delay below d.
func (t *Transport) jitter(d time.Duration) time.Duration {
	if t.Jitter != nil {
		return t.Jitter(d)
	}
	if d <= 0 {
		return 0
	}
	return rand.N(d) //nolint:gosec // jitter does not need a secure source
}

// pause stops every request until d from now, extending any current pause.
func (t *Transport) pause(d time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	until := t.now().Add(d)
	extended := until.After(t.pausedUntil)
	if extended {
		t.pausedUntil = until
	}
	t.pauses.Add(1)
	// Only a new or longer pause is news; workers hitting the same limit
	// at once would otherwise each report it.
	if extended && t.OnWait != nil {
		t.OnWait("secondary rate limit", d)
	}
}

// waitForPause blocks while a secondary rate limit pause is in effect.
func (t *Transport) waitForPause(ctx context.Context) error {
	for {
		t.mu.Lock()
		d := t.pausedUntil.Sub(t.now())
		t.mu.Unlock()
		if d <= 0 {
			return nil
		}
		if err := t.sleep(ctx, d); err != nil {
			return err
		}
	}
}

// checkQuota refuses a request when its resource is inside the headroom.
// Search quotas reset every minute, so for them it waits instead.
func (t *Transport) checkQuota(ctx context.Context, resource string) error {
	t.mu.Lock()
	q, ok := t.quotas[resource]
	t.mu.Unlock()
	if !ok || q.Limit == 0 || !q.Reset.After(t.now()) {
		return nil
	}
	search := strings.HasSuffix(resource, "search")
	reserve := int(float64(q.Limit) * t.Headroom)
	if search {
		reserve = 0
	}
	if q.Remaining > reserve {
		return nil
	}
	if search {
		if t.OnWait != nil {
			t.OnWait(resource+" rate limit", q.Reset.Sub(t.now()))
		}
		err := t.sleep(ctx, q.Reset.Sub(t.now()))
		t.mu.Lock()
		delete(t.quotas, resource)
		t.mu.Unlock()
		return err
	}
	return &QuotaError{Resource: resource, Remaining: q.Remaining, Limit: q.Limit, Reset: q.Reset}
}

// recordQuota stores the X-RateLimit headers of a response.
func (t *Transport) recordQuota(h http.Header) {
	limit, err1 := strconv.Atoi(h.Get("X-RateLimit-Limit"))
	remaining, err2 := strconv.Atoi(h.Get("X-RateLimit-Remaining"))
	reset, err3 := strconv.ParseInt(h.Get("X-RateLimit-Reset"), 10, 64)
	if err1 != nil || err2 != nil || err3 != nil {
		return
	}
	res := h.Get("X-RateLimit-Resource")
	if res == "" {
		res = "core"
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.quotas == nil {
		t.quotas = map[string]RateLimit{}
	}
	t.quotas[res] = RateLimit{Resource: res, Limit: limit, Remaining: remaining, Reset: time.Unix(reset, 0)}
}

// limitKind is what a response says about rate limits.
type limitKind int

const (
	notLimited limitKind = iota
	// secondary is a secondary rate limit: retry after a wait.
	secondary
	// spentQuota is a spent primary quota: nothing until it resets.
	spentQuota
)

// limit is how a response is rate limited.
type limit struct {
	kind limitKind
	// wait is how long to wait before retrying; negative when GitHub
	// didn't say.
	wait time.Duration
	// quota describes a spent primary quota.
	quota *QuotaError
}

// classify reads a response the way GitHub's docs say to: a Retry-After
// header is a secondary limit with its wait; otherwise no remaining quota
// means wait until the reset; otherwise a 429, or a 403 that says so, is a
// secondary limit with no wait given. GraphQL reports spent quota as a
// 200 with a RATE_LIMITED error. The body is read only when it can tell.
func classify(resp *http.Response, resource string, now time.Time) (limit, error) {
	code := resp.StatusCode
	graphql := resource == "graphql" && code == http.StatusOK && resp.Header.Get("X-RateLimit-Remaining") == "0"
	if code != http.StatusForbidden && code != http.StatusTooManyRequests && !graphql {
		return limit{}, nil
	}
	if !graphql {
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			return limit{kind: secondary, wait: retryAfter(ra, now)}, nil
		}
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	resp.Body = io.NopCloser(bytes.NewReader(body))
	if err != nil {
		return limit{}, err
	}
	if resp.Header.Get("X-RateLimit-Remaining") == "0" {
		if graphql && !bytes.Contains(body, []byte("RATE_LIMITED")) {
			return limit{}, nil
		}
		q := &QuotaError{Resource: resource, Reset: now.Add(defaultRetryAfter)}
		q.Limit, _ = strconv.Atoi(resp.Header.Get("X-RateLimit-Limit"))
		if reset, err := strconv.ParseInt(resp.Header.Get("X-RateLimit-Reset"), 10, 64); err == nil {
			q.Reset = time.Unix(reset, 0)
		}
		return limit{kind: spentQuota, wait: max(q.Reset.Sub(now), time.Second), quota: q}, nil
	}
	lower := bytes.ToLower(body)
	if code == http.StatusTooManyRequests || bytes.Contains(lower, []byte("secondary rate limit")) || bytes.Contains(lower, []byte("abuse")) {
		return limit{kind: secondary, wait: -1}, nil
	}
	return limit{}, nil
}

// retryAfter parses a Retry-After header: seconds, or an HTTP date.
func retryAfter(ra string, now time.Time) time.Duration {
	if secs, err := strconv.Atoi(ra); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if at, err := http.ParseTime(ra); err == nil {
		return max(at.Sub(now), 0)
	}
	return defaultRetryAfter
}

// isRetryableStatus reports server errors worth retrying.
func isRetryableStatus(code int) bool {
	switch code {
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	}
	return false
}

// isWrite reports whether a request creates or changes content. GraphQL
// requests count as writes only when they are mutations.
func isWrite(method, path string, body []byte) bool {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodOptions:
		return false
	}
	if strings.HasSuffix(path, "/graphql") {
		return isMutation(body)
	}
	return true
}

// isMutation reports whether a GraphQL request body holds a mutation.
func isMutation(body []byte) bool {
	i := bytes.Index(body, []byte(`"query"`))
	if i < 0 {
		return false
	}
	rest := bytes.TrimLeft(body[i+len(`"query"`):], " \t\r\n:")
	rest = bytes.TrimLeft(rest, `"`)
	rest = bytes.TrimLeft(rest, ` \t\r\n`)
	for bytes.HasPrefix(rest, []byte(`\n`)) || bytes.HasPrefix(rest, []byte(`\t`)) {
		rest = bytes.TrimLeft(rest[2:], " ")
	}
	return bytes.HasPrefix(rest, []byte("mutation"))
}

// createsContent reports whether a request creates content, which GitHub
// limits to ContentPerMinute and ContentPerHour: a new pull request, issue,
// comment or review. Other writes, such as labels, review requests, edits,
// merges and branch deletions, only get the per-second write pacing.
func createsContent(method, path string) bool {
	if method != http.MethodPost {
		return false
	}
	for _, suffix := range []string{"/pulls", "/issues", "/comments", "/reviews"} {
		if strings.HasSuffix(path, suffix) {
			return true
		}
	}
	return false
}

// resourceFor maps a request path to its rate limit resource name.
func resourceFor(path string) string {
	switch {
	case strings.HasSuffix(path, "/graphql"):
		return "graphql"
	case strings.Contains(path, "/search/code"):
		return "code_search"
	case strings.Contains(path, "/search/"):
		return "search"
	default:
		return "core"
	}
}

// drain discards and closes a response body so the connection is reused.
func drain(resp *http.Response) {
	_, _ = io.Copy(io.Discard, resp.Body)
	_ = resp.Body.Close()
}
