//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"business/internal/library/oswrapper"
	"business/internal/library/ratelimit"
	config "business/internal/library/ratelimit/config"
	redisclient "business/internal/library/redis"
	"business/internal/library/timewrapper"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisRateLimiter runs integration tests against a real Redis instance.
// To run these tests, set the Redis connection env vars (HOST/PORT/PASSWORD/DB)
// and execute: go test -tags integration ./test -run TestRedisRateLimiter
var redisEnvTestMu sync.Mutex

func TestRedisRateLimiter(t *testing.T) {
	t.Parallel()
	redisEnvTestMu.Lock()
	defer redisEnvTestMu.Unlock()
	envVars := loadRedisConnectionEnv()
	redisEnvWrapper := newOverrideOsWrapper(envVars)
	redisclient.SetDefaultOsWrapper(redisEnvWrapper)
	t.Cleanup(func() {
		redisclient.SetDefaultOsWrapper(nil)
	})

	configEnv := newConfigEnvManager()
	config.SetEnvGetter(configEnv.get)
	t.Cleanup(func() {
		config.SetEnvGetter(nil)
	})

	client, err := redisclient.New(redisclient.Config{}, redisEnvWrapper, nil)
	require.NoError(t, err)

	ctx := context.Background()
	rawRedis := newRawRedisClient(t, envVars)
	t.Cleanup(func() { _ = rawRedis.Close() })
	clock := timewrapper.NewClock()
	oswReal := oswrapper.New(nil)

	// Verify Redis connectivity
	require.NoError(t, rawRedis.Ping(ctx).Err(), "failed to connect to Redis")

	t.Run("enforces 1 second window limit", func(t *testing.T) {
		// Clear any existing keys
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		// Set strict window: 1s/3req
		restore := configEnv.set("REDIS_RATE_LIMIT_WINDOW_CONFIG", "1:3")
		defer restore()

		// Create a fresh limiter
		limiter := createRedisLimiter(t, client, clock, oswReal, "gmail")

		// First 3 requests should succeed
		for i := 0; i < 3; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		// 4th request should block and wait for window to expire
		// Use a context with timeout to verify it waits approximately 1 second
		start := time.Now()
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		// Should succeed after waiting ~1 second for window to expire
		assert.NoError(t, err, "request should succeed after waiting")
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should wait at least ~1s")
		assert.Less(t, elapsed, 1500*time.Millisecond, "should not wait much longer than 1s")
	})

	t.Run("enforces 10 second window limit", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "openai", "global")

		restore := configEnv.set("REDIS_RATE_LIMIT_WINDOW_CONFIG", "1:100,10:5")
		defer restore()

		limiter := createRedisLimiter(t, client, clock, oswReal, "openai")

		// Make 5 requests quickly
		for i := 0; i < 5; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		// 6th request should block and wait for 10s window to expire
		// Use a short timeout to verify it attempts to wait (but we don't actually wait 10s in test)
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		// Should fail with context deadline exceeded since 10s > 500ms
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		// Should have waited close to the timeout duration
		assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond)
		assert.Less(t, elapsed, 700*time.Millisecond)
	})

	t.Run("respects context cancellation during wait", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		restore := configEnv.set("REDIS_RATE_LIMIT_WINDOW_CONFIG", "1:2")
		defer restore()

		limiter := createRedisLimiter(t, client, clock, oswReal, "gmail")

		// Exhaust the limit
		for i := 0; i < 2; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		// 3rd request would need to wait, but we cancel the context
		ctxWithCancel, cancel := context.WithCancel(ctx)

		// Cancel after a short delay to allow Wait to enter its waiting state
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := limiter.Wait(ctxWithCancel)
		elapsed := time.Since(start)

		// Should fail with context canceled, not wait the full window
		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, elapsed, 500*time.Millisecond, "should stop quickly on cancel")
	})

	t.Run("returns error when Redis is unavailable", func(t *testing.T) {
		invalidWrapper := newOverrideOsWrapper(map[string]string{
			"REDIS_HOST": "invalid-host",
			"REDIS_PORT": "9999",
			"REDIS_DB":   "0",
		})
		redisclient.SetDefaultOsWrapper(invalidWrapper)
		defer redisclient.SetDefaultOsWrapper(redisEnvWrapper)

		// Create limiter with invalid Redis URL
		limiter := createRedisLimiter(t, client, clock, oswReal, "gmail")

		err := limiter.Wait(context.Background())
		assert.Error(t, err)

		var redisErr *ratelimit.ErrRedisUnavailable
		assert.ErrorAs(t, err, &redisErr)
	})

	t.Run("handles multiple windows correctly", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		// Windows: 1s/10req, 10s/20req, 60s/100req
		restore := configEnv.set("REDIS_RATE_LIMIT_WINDOW_CONFIG", "1:10,10:20,60:100")
		defer restore()

		limiter := createRedisLimiter(t, client, clock, oswReal, "gmail")

		// Make 10 requests (should hit 1s limit)
		for i := 0; i < 10; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		// 11th request should wait ~1s for window to expire
		start := time.Now()
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		assert.NoError(t, err, "request should succeed after waiting")
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should wait at least ~1s")
		assert.Less(t, elapsed, 1500*time.Millisecond, "should not wait much longer than 1s")
	})

	t.Run("uses default windows when config not provided", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		restoreRPS := configEnv.set("REDIS_RATE_LIMIT_RPS", "5")
		defer restoreRPS()
		restoreWindows := configEnv.set("REDIS_RATE_LIMIT_WINDOW_CONFIG", "")
		defer restoreWindows()

		limiter := createRedisLimiter(t, client, clock, oswReal, "gmail")

		// Default windows should be: 1s/5, 10s/25, 60s/150
		// Make 5 requests
		for i := 0; i < 5; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		// 6th should wait ~1s for window to expire
		start := time.Now()
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		assert.NoError(t, err, "request should succeed after waiting")
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should wait at least ~1s")
		assert.Less(t, elapsed, 1500*time.Millisecond, "should not wait much longer than 1s")
	})
}

// createRedisLimiter creates a new RedisLimiter instance for testing.
func createRedisLimiter(t *testing.T, client redisclient.ClientInterface, clock timewrapper.ClockInterface, osw oswrapper.OsWapperInterface, namespace string) ratelimit.Limiter {
	t.Helper()

	provider := ratelimit.NewProvider(client, clock, osw, nil)
	if namespace == "gmail" {
		return provider.GetGmailLimiter()
	}
	return provider.GetOpenAILimiter()
}

func newRawRedisClient(t *testing.T, env map[string]string) *goredis.Client {
	t.Helper()

	db, err := strconv.Atoi(env["REDIS_DB"])
	require.NoError(t, err, "invalid REDIS_DB value")

	client := goredis.NewClient(&goredis.Options{
		Addr:     fmt.Sprintf("%s:%s", env["REDIS_HOST"], env["REDIS_PORT"]),
		Password: env["REDIS_PASSWORD"],
		DB:       db,
	})
	return client
}

// cleanupRedisKeys removes all rate limit keys for a given namespace/bucket.
func cleanupRedisKeys(t *testing.T, client *goredis.Client, namespace, bucket string) {
	t.Helper()

	ctx := context.Background()
	pattern := "ratelimit:" + namespace + ":" + bucket + ":*"

	var cursor uint64
	for {
		keys, nextCursor, err := client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			t.Logf("warning: scan iteration error: %v", err)
			return
		}
		for _, key := range keys {
			if err := client.Del(ctx, key).Err(); err != nil {
				t.Logf("warning: failed to delete key %s: %v", key, err)
			}
		}
		if nextCursor == 0 {
			break
		}
		cursor = nextCursor
	}
}

func loadRedisConnectionEnv() map[string]string {
	return map[string]string{
		"REDIS_HOST":     envOrDefault("REDIS_HOST", "localhost"),
		"REDIS_PORT":     envOrDefault("REDIS_PORT", "6379"),
		"REDIS_PASSWORD": os.Getenv("REDIS_PASSWORD"),
		"REDIS_DB":       envOrDefault("REDIS_DB", "0"),
	}
}

type overrideOsWrapper struct {
	base oswrapper.OsWapperInterface
	vars map[string]string
}

func newOverrideOsWrapper(vars map[string]string) oswrapper.OsWapperInterface {
	return &overrideOsWrapper{
		base: oswrapper.New(nil),
		vars: vars,
	}
}

func (o *overrideOsWrapper) ReadFile(path string) (string, error) {
	return o.base.ReadFile(path)
}

func (o *overrideOsWrapper) GetEnv(key string) (string, error) {
	if v, ok := o.vars[key]; ok {
		if v == "" {
			return "", fmt.Errorf("environment variable %s not set", key)
		}
		return v, nil
	}
	return o.base.GetEnv(key)
}

func envOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}

type configEnvManager struct {
	mu   sync.RWMutex
	vars map[string]string
}

func newConfigEnvManager() *configEnvManager {
	return &configEnvManager{
		vars: make(map[string]string),
	}
}

func (c *configEnvManager) get(key string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.vars[key]
}

func (c *configEnvManager) set(key, value string) func() {
	c.mu.Lock()
	prev, existed := c.vars[key]
	if value == "" {
		delete(c.vars, key)
	} else {
		c.vars[key] = value
	}
	c.mu.Unlock()

	return func() {
		c.mu.Lock()
		defer c.mu.Unlock()
		if existed {
			c.vars[key] = prev
		} else {
			delete(c.vars, key)
		}
	}
}
