package redislimit

import (
	"context"
	"fmt"
	"time"

	"business/internal/library/logger"
	"business/internal/library/ratelimit/config"
	redisclient "business/internal/library/redis"
	"business/internal/library/timewrapper"
)

// Limiter implements distributed rate limiting using Redis sliding windows.
// It uses the Redis client's RunRateLimitScript method to execute rate limiting logic.
type Limiter struct {
	client    redisclient.ClientInterface
	clock     timewrapper.ClockInterface
	namespace string
	bucket    string
	windows   []config.Window
	log       logger.Interface
}

// NewLimiter creates a new Redis-backed rate limiter for the provided namespace.
// It accepts a Redis client and clock via dependency injection.
func NewLimiter(
	client redisclient.ClientInterface,
	clock timewrapper.ClockInterface,
	namespace string,
	log logger.Interface,
) *Limiter {
	if log == nil {
		log = logger.NewNop()
	}
	log = log.With(logger.String("component", "redis_limiter"))

	windows := config.Windows(namespace)

	return &Limiter{
		client:    client,
		clock:     clock,
		namespace: namespace,
		bucket:    "global",
		windows:   windows,
		log:       log,
	}
}

// Wait blocks until the limiter allows the next request or the context is canceled.
func (r *Limiter) Wait(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Guard against nil client
	if r.client == nil {
		return &redisclient.ErrRedisUnavailable{
			Err: fmt.Errorf("redis client not initialized"),
		}
	}

	for {
		// Build parameters for rate limit script
		params := redisclient.RateLimitParams{
			Namespace: r.namespace,
			Bucket:    r.bucket,
			Time:      r.clock.Now(),
			Windows:   r.windows,
		}

		// Execute rate limit script via Redis client
		result, err := r.client.RunRateLimitScript(ctx, params)
		if err != nil {
			// Error is already logged and wrapped by RunRateLimitScript
			return err
		}

		// If allowed, return immediately
		if result.Allowed {
			return nil
		}

		// Rate limit exceeded - log and wait
		waitDuration := int64(result.WindowSeconds)

		rateLimitErr := &ErrRateLimitExceeded{
			WindowSeconds: result.WindowSeconds,
			Limit:         result.Limit,
			Current:       result.Current,
		}
		r.log.Warn("rate limiter triggered",
			logger.String("namespace", r.namespace),
			logger.String("bucket", r.bucket),
			logger.Int("window_seconds", result.WindowSeconds),
			logger.Int("limit", result.Limit),
			logger.Int("current", result.Current),
			logger.Int("wait_seconds", int(waitDuration)),
			logger.Err(rateLimitErr),
		)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-r.clock.After(time.Duration(waitDuration) * time.Second):
		}
	}
}
