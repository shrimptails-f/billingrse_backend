package ratelimit

import "context"

// Limiter defines the interface for rate limiting implementations.
// Implementations may use in-memory tokens, Redis sliding windows, or other strategies.
type Limiter interface {
	// Wait blocks until the limiter allows the next request or the context is canceled.
	// Returns an error if Redis is unavailable (fail-closed) or context is done.
	// For rate limit exceeded cases, Wait will block and retry until a request can be allowed.
	Wait(ctx context.Context) error
}
