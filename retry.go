package mlx

import (
	"context"
	"math"
	"math/rand"
	"net/http"
	"time"
)

const (
	defaultRetryMaxAttempts     = 3
	defaultRetryInitialInterval = 500 * time.Millisecond
	defaultRetryMaxInterval     = 5 * time.Second
	defaultRetryMultiplier      = 2.0
	defaultRetryJitter          = 0.2
)

// RetryOptions controls transport-level retry/backoff behavior.
type RetryOptions struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	Jitter          float64
	ShouldRetry     func(error) bool
	Rand            *rand.Rand
}

func normalizeRetryOptions(opts RetryOptions) RetryOptions {
	if opts.MaxAttempts <= 0 {
		opts.MaxAttempts = defaultRetryMaxAttempts
	}
	if opts.InitialInterval <= 0 {
		opts.InitialInterval = defaultRetryInitialInterval
	}
	if opts.MaxInterval <= 0 {
		opts.MaxInterval = defaultRetryMaxInterval
	}
	if opts.MaxInterval < opts.InitialInterval {
		opts.MaxInterval = opts.InitialInterval
	}
	if opts.Multiplier <= 1 {
		opts.Multiplier = defaultRetryMultiplier
	}
	if opts.Jitter < 0 {
		opts.Jitter = 0
	}
	if opts.ShouldRetry == nil {
		opts.ShouldRetry = IsRetryableError
	}
	if opts.Rand == nil {
		opts.Rand = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return opts
}

func nextRetryInterval(current time.Duration, multiplier float64, max time.Duration) time.Duration {
	next := time.Duration(math.Round(float64(current) * multiplier))
	if next < current {
		next = current
	}
	if next > max {
		return max
	}
	return next
}

func jitterDuration(base time.Duration, jitter float64, r *rand.Rand) time.Duration {
	if base <= 0 || jitter <= 0 || r == nil {
		return base
	}
	factor := 1 + ((r.Float64()*2 - 1) * jitter)
	if factor < 0 {
		factor = 0
	}
	return time.Duration(float64(base) * factor)
}

func retryDelay(err error, fallback time.Duration, opts RetryOptions) time.Duration {
	if ra := RetryAfter(err); ra > 0 {
		if ra > opts.MaxInterval {
			return opts.MaxInterval
		}
		return ra
	}
	return jitterDuration(fallback, opts.Jitter, opts.Rand)
}

func doWithRetry(ctx context.Context, opts RetryOptions, fn func() (*http.Response, error)) (*http.Response, error) {
	opts = normalizeRetryOptions(opts)
	interval := opts.InitialInterval

	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= opts.MaxAttempts; attempt++ {
		resp, err := fn()
		if err == nil {
			return resp, nil
		}
		lastResp = resp
		lastErr = err
		if attempt == opts.MaxAttempts || !opts.ShouldRetry(err) {
			return lastResp, lastErr
		}
		if err := waitWithContext(ctx, retryDelay(err, interval, opts)); err != nil {
			return lastResp, err
		}
		interval = nextRetryInterval(interval, opts.Multiplier, opts.MaxInterval)
	}

	return lastResp, lastErr
}
