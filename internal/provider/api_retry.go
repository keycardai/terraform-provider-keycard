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
// for up to a minute. The retry window must cover both.
const (
	apiRetryWindow       = 2 * time.Minute
	apiRetryInitialDelay = 2 * time.Second
	apiRetryMaxDelay     = 10 * time.Second
)

// callWithRetry executes call, retrying with backoff while retryable
// returns true, until the retry window elapses or ctx is cancelled.
// Transport errors are returned immediately; when the window closes the
// last response is returned for the caller to handle like any other.
func callWithRetry[R apiResponse](ctx context.Context, call func() (R, error), retryable func(R) bool) (R, error) {
	deadline := time.Now().Add(apiRetryWindow)
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

// retryOnNotFound retries 404 responses. Use only on calls where a 404
// can only mean the containing scope has not finished provisioning (e.g.
// zone-scoped list endpoints), never where it is a real lookup miss.
func retryOnNotFound[R apiResponse](resp R) bool {
	return resp.StatusCode() == 404
}
