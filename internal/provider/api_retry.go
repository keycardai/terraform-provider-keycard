package provider

import (
	"context"
	"time"
)

// apiResponse is implemented by every generated *WithResponse return type.
type apiResponse interface {
	StatusCode() int
}

// Zones are provisioned across services asynchronously after creation, and
// a request that races provisioning can be negatively cached server-side
// for up to a minute. The default retry window must cover both.
const (
	apiRetryWindow       = 2 * time.Minute
	apiRetryInitialDelay = 2 * time.Second
	apiRetryMaxDelay     = 10 * time.Second

	// notFoundRetryWindow is the default window for paths where a persistent 404 is
	// a genuine miss (the requested entity does not exist) rather than provisioning
	// lag. It covers replica propagation for a freshly written entity without
	// making a real miss wait out the full apiRetryWindow.
	notFoundRetryWindow = 30 * time.Second
)

// retryOption overrides a callWithRetry default.
type retryOption func(*time.Duration)

// withRetryWindow bounds how long callWithRetry keeps retrying. Use a shorter
// window on paths where a retryable status is usually a genuine miss rather
// than provisioning lag, so real misses surface without waiting out the full
// default window.
func withRetryWindow(window time.Duration) retryOption {
	return func(w *time.Duration) { *w = window }
}

// callWithRetry executes call, retrying with backoff while retryable
// returns true, until the retry window elapses or ctx is cancelled.
// Transport errors are returned immediately; when the window closes the
// last response is returned for the caller to handle like any other.
func callWithRetry[R apiResponse](ctx context.Context, call func() (R, error), retryable func(R) bool, opts ...retryOption) (R, error) {
	window := apiRetryWindow
	for _, opt := range opts {
		opt(&window)
	}

	deadline := time.Now().Add(window)
	delay := apiRetryInitialDelay

	for {
		resp, err := call()
		if err != nil || !retryable(resp) || time.Now().After(deadline) {
			return resp, err
		}

		select {
		case <-ctx.Done():
			return resp, ctx.Err()
		case <-time.After(delay):
		}

		delay = min(delay*2, apiRetryMaxDelay)
	}
}

// retryOnNotFound retries 404 responses. Use on calls where a 404 may be
// transient (e.g. a scope still provisioning or an eventually-consistent
// replica lagging). Genuine misses still surface, just after the retry
// window elapses, so keep that window short where a real miss is likely.
func retryOnNotFound[R apiResponse](resp R) bool {
	return resp.StatusCode() == 404
}

// retryOnNotFoundOrConflict retries 404 and 412 responses. Use on If-Match
// updates against an eventually-consistent backend: a 412 usually means the
// ETag in state is stale (the refresh read a lagging replica), not a genuine
// concurrent modification, so the caller should refetch the ETag between
// attempts and let the write self-heal. A truly stuck 412 still surfaces once
// the retry window elapses.
func retryOnNotFoundOrConflict[R apiResponse](resp R) bool {
	return resp.StatusCode() == 404 || resp.StatusCode() == 412
}
