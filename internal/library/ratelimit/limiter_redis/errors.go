package redislimit

import (
	"fmt"

	redisclient "business/internal/library/redis"
)

// ErrRateLimitExceeded is returned when a request exceeds the configured rate limit.
type ErrRateLimitExceeded struct {
	WindowSeconds int
	Limit         int
	Current       int
}

func (e *ErrRateLimitExceeded) Error() string {
	return fmt.Sprintf("rate limit exceeded: %d requests in %d second window (limit: %d)",
		e.Current, e.WindowSeconds, e.Limit)
}

// ErrRedisUnavailable indicates Redis backend is unavailable (fail-closed).
// This type is re-exported from the redis package for backward compatibility.
type ErrRedisUnavailable = redisclient.ErrRedisUnavailable
