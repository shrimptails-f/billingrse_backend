package retry

import (
	"context"
	"time"
)

var (
	// DefaultBackoff configures the wait durations between attempts.
	DefaultBackoff = []time.Duration{
		2 * time.Second,
		3 * time.Second,
		5 * time.Second,
	}
)

// ShouldRetryFunc is a function type that determines whether an error should be retried.
// Return true to retry the operation, false to stop immediately.
type ShouldRetryFunc func(error) bool

// Do executes the given operation with retry semantics.
// The operation is attempted len(backoff)+1 times.
// This function retries all errors unconditionally for backward compatibility.
func Do(ctx context.Context, backoff []time.Duration, operation func(context.Context) error) error {
	return DoWithCondition(ctx, backoff, nil, operation)
}

// DoWithCondition executes the given operation with conditional retry semantics.
// The operation is attempted up to len(backoff)+1 times.
// If backoff is nil or empty, the operation is attempted exactly once (no retries).
// If shouldRetry is nil, all errors are retried (same as Do).
// If shouldRetry returns false for an error, the retry loop stops immediately.
func DoWithCondition(ctx context.Context, backoff []time.Duration, shouldRetry ShouldRetryFunc, operation func(context.Context) error) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// If backoff is nil or empty, attempt exactly once (no retries)
	attempts := 1
	if len(backoff) > 0 {
		attempts = len(backoff) + 1
	}

	var lastErr error
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			delay := backoff[attempt-1]
			if err := waitWithContext(ctx, delay); err != nil {
				return err
			}
		}

		if err := operation(ctx); err != nil {
			lastErr = err
			// If shouldRetry is provided and returns false, stop retrying
			if shouldRetry != nil && !shouldRetry(err) {
				return lastErr
			}
			continue
		}
		return nil
	}

	return lastErr
}

func waitWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
