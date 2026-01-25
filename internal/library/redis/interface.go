package redisclient

import (
	"context"
	"time"

	"business/internal/library/ratelimit/config"
	"business/internal/library/redis/script"
)

// RateLimitParams holds the parameters for rate limit script execution.
type RateLimitParams struct {
	Namespace string
	Bucket    string
	Time      time.Time
	Windows   []config.Window
}

// RateLimitResult holds the result of rate limit script execution.
type RateLimitResult struct {
	Allowed       bool
	WindowSeconds int
	Limit         int
	Current       int
}

// ClientInterface abstracts the Redis client operations needed for rate limiting.
type ClientInterface interface {
	EvalScript(ctx context.Context, scr script.Script, keys []string, args ...interface{}) (interface{}, error)
	EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error)
	RunRateLimitScript(ctx context.Context, params RateLimitParams) (RateLimitResult, error)
}
