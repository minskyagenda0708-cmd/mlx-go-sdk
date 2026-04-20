package mlx

import (
	"context"
	"fmt"
	"math"
	"time"
)

const (
	defaultPollInitialInterval = 2 * time.Second
	defaultPollMaxInterval     = 10 * time.Second
	defaultPollTimeout         = 2 * time.Minute
	defaultPollMultiplier      = 1.5
)

// PollOptions controls retry/backoff behavior for launcher polling helpers.
type PollOptions struct {
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Timeout         time.Duration
	Multiplier      float64
}

func normalizePollOptions(opts PollOptions) PollOptions {
	if opts.InitialInterval <= 0 {
		opts.InitialInterval = defaultPollInitialInterval
	}
	if opts.MaxInterval <= 0 {
		opts.MaxInterval = defaultPollMaxInterval
	}
	if opts.MaxInterval < opts.InitialInterval {
		opts.MaxInterval = opts.InitialInterval
	}
	if opts.Timeout <= 0 {
		opts.Timeout = defaultPollTimeout
	}
	if opts.Multiplier <= 1 {
		opts.Multiplier = defaultPollMultiplier
	}
	return opts
}

func nextPollInterval(current time.Duration, multiplier float64, max time.Duration) time.Duration {
	next := time.Duration(math.Round(float64(current) * multiplier))
	if next < current {
		next = current
	}
	if next > max {
		return max
	}
	return next
}

func waitWithContext(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func pollUntil[T any](ctx context.Context, opts PollOptions, description string, check func(context.Context) (T, *Response, error), ready func(T) bool, statusText func(T) string) (T, *Response, error) {
	var zero T
	opts = normalizePollOptions(opts)
	deadline := time.Now().Add(opts.Timeout)
	interval := opts.InitialInterval

	var lastValue T
	var lastResp *Response
	var lastErr error

	for {
		value, resp, err := check(ctx)
		if err == nil && ready(value) {
			return value, resp, nil
		}

		lastValue = value
		lastResp = resp
		lastErr = err

		if err := ctx.Err(); err != nil {
			return zero, lastResp, err
		}
		if !time.Now().Before(deadline) {
			break
		}
		if err := waitWithContext(ctx, interval); err != nil {
			return zero, lastResp, err
		}
		interval = nextPollInterval(interval, opts.Multiplier, opts.MaxInterval)
	}

	if lastErr != nil {
		return zero, lastResp, fmt.Errorf("%s before timeout: %w", description, lastErr)
	}
	status := statusText(lastValue)
	if status != "" {
		return zero, lastResp, fmt.Errorf("%s before timeout, last status=%s", description, status)
	}
	return zero, lastResp, fmt.Errorf("%s before timeout", description)
}
