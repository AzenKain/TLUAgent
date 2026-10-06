package llm

import (
	"math/rand"
	"net/http"
	"time"

	"tluagent-web/pkg/netx"
)

// RetryableRoundTripper wraps an http.RoundTripper with exponential backoff retry.
type RetryableRoundTripper struct {
	Transport   http.RoundTripper
	MaxRetries  int
	InitialWait time.Duration
	MaxWait     time.Duration
}

// NewRetryClient creates an http.Client with automatic exponential backoff for 429 and 5xx errors.
func NewRetryClient(timeout time.Duration, maxRetries int) *http.Client {
	return NewSafeRetryClient(timeout, maxRetries, false)
}

// NewSafeRetryClient creates an http.Client with configurable private network permission and retry backoff.
func NewSafeRetryClient(timeout time.Duration, maxRetries int, allowPrivate bool) *http.Client {
	return NewCustomRetryClient(timeout, maxRetries, 500*time.Millisecond, 5*time.Second, allowPrivate)
}

// NewCustomRetryClient creates an http.Client with custom retry backoff and private network configuration.
func NewCustomRetryClient(timeout time.Duration, maxRetries int, initialWait, maxWait time.Duration, allowPrivate bool) *http.Client {
	if maxRetries < 0 {
		maxRetries = 0
	}
	if initialWait <= 0 {
		initialWait = 500 * time.Millisecond
	}
	if maxWait <= 0 {
		maxWait = 5 * time.Second
	}
	transport := netx.NewSafeTransport(allowPrivate)
	return &http.Client{
		Timeout: timeout,
		Transport: &RetryableRoundTripper{
			Transport:   transport,
			MaxRetries:  maxRetries,
			InitialWait: initialWait,
			MaxWait:     maxWait,
		},
	}
}

func (rt *RetryableRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	wait := rt.InitialWait
	for attempt := 0; attempt <= rt.MaxRetries; attempt++ {
		resp, err = rt.Transport.RoundTrip(req)
		if err == nil && !isRetryableStatus(resp.StatusCode) {
			return resp, nil
		}

		if attempt == rt.MaxRetries {
			break
		}

		if req.Context().Err() != nil {
			if resp != nil {
				resp.Body.Close()
			}
			return nil, req.Context().Err()
		}

		if resp != nil {
			resp.Body.Close()
		}

		jitter := time.Duration(rand.Int63n(int64(wait / 2)))
		sleepDuration := wait + jitter
		if sleepDuration > rt.MaxWait {
			sleepDuration = rt.MaxWait
		}

		select {
		case <-time.After(sleepDuration):
		case <-req.Context().Done():
			return nil, req.Context().Err()
		}

		wait *= 2
	}

	return resp, err
}

func isRetryableStatus(status int) bool {
	return status == http.StatusTooManyRequests ||
		status == http.StatusBadGateway ||
		status == http.StatusServiceUnavailable ||
		status == http.StatusGatewayTimeout
}
