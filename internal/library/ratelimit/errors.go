package ratelimit

import redislimit "business/internal/library/ratelimit/limiter_redis"

// ErrRateLimitExceeded is an alias to the Redis limiter error for compatibility.
type ErrRateLimitExceeded = redislimit.ErrRateLimitExceeded

// ErrRedisUnavailable is an alias to the Redis limiter error for compatibility.
type ErrRedisUnavailable = redislimit.ErrRedisUnavailable
