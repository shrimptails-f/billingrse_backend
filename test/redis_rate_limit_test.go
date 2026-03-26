//go:build integration
// +build integration

package test

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"business/internal/library/ratelimit"
	config "business/internal/library/ratelimit/config"
	redislimit "business/internal/library/ratelimit/limiter_redis"
	redisclient "business/internal/library/redis"
	mocklibrary "business/test/mock/library"

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
	redisEnvWrapper := newRedisEnvWrapper(t, envVars)
	redisclient.SetDefaultOsWrapper(redisEnvWrapper)
	t.Cleanup(func() {
		redisclient.SetDefaultOsWrapper(nil)
	})

	client, err := redisclient.New(redisclient.Config{}, redisEnvWrapper, nil)
	require.NoError(t, err)

	ctx := context.Background()
	rawRedis := newRawRedisClient(t, envVars)
	t.Cleanup(func() { _ = rawRedis.Close() })

	require.NoError(t, rawRedis.Ping(ctx).Err(), "failed to connect to Redis")

	t.Run("enforces 1 second window limit", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		limiter := createRedisLimiter(t, client, "gmail", []config.Window{
			{SizeSeconds: 1, Limit: 3},
		})

		for i := 0; i < 3; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		start := time.Now()
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		assert.NoError(t, err, "request should succeed after waiting")
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should wait at least ~1s")
		assert.Less(t, elapsed, 1500*time.Millisecond, "should not wait much longer than 1s")
	})

	t.Run("enforces 10 second window limit", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "openai", "global")

		limiter := createRedisLimiter(t, client, "openai", []config.Window{
			{SizeSeconds: 1, Limit: 100},
			{SizeSeconds: 10, Limit: 5},
		})

		for i := 0; i < 5; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		ctxWithTimeout, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
		defer cancel()

		start := time.Now()
		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.DeadlineExceeded)
		assert.GreaterOrEqual(t, elapsed, 400*time.Millisecond)
		assert.Less(t, elapsed, 700*time.Millisecond)
	})

	t.Run("respects context cancellation during wait", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		limiter := createRedisLimiter(t, client, "gmail", []config.Window{
			{SizeSeconds: 1, Limit: 2},
		})

		for i := 0; i < 2; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		ctxWithCancel, cancel := context.WithCancel(ctx)
		go func() {
			time.Sleep(100 * time.Millisecond)
			cancel()
		}()

		start := time.Now()
		err := limiter.Wait(ctxWithCancel)
		elapsed := time.Since(start)

		assert.Error(t, err)
		assert.ErrorIs(t, err, context.Canceled)
		assert.Less(t, elapsed, 500*time.Millisecond, "should stop quickly on cancel")
	})

	t.Run("returns error when Redis is unavailable", func(t *testing.T) {
		invalidWrapper := newRedisEnvWrapper(t, map[string]string{
			"REDIS_HOST":     "invalid-host",
			"REDIS_PORT":     "9999",
			"REDIS_PASSWORD": envVars["REDIS_PASSWORD"],
			"REDIS_DB":       "0",
		})
		redisclient.SetDefaultOsWrapper(invalidWrapper)
		defer redisclient.SetDefaultOsWrapper(redisEnvWrapper)

		invalidClient, err := redisclient.New(redisclient.Config{}, invalidWrapper, nil)
		require.NoError(t, err)

		limiter := createRedisLimiter(t, invalidClient, "gmail", []config.Window{
			{SizeSeconds: 1, Limit: 3},
		})

		err = limiter.Wait(context.Background())
		assert.Error(t, err)

		var redisErr *ratelimit.ErrRedisUnavailable
		assert.ErrorAs(t, err, &redisErr)
	})

	t.Run("handles multiple windows correctly", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		limiter := createRedisLimiter(t, client, "gmail", []config.Window{
			{SizeSeconds: 1, Limit: 10},
			{SizeSeconds: 10, Limit: 20},
			{SizeSeconds: 60, Limit: 100},
		})

		for i := 0; i < 10; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

		start := time.Now()
		ctxWithTimeout, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()

		err := limiter.Wait(ctxWithTimeout)
		elapsed := time.Since(start)

		assert.NoError(t, err, "request should succeed after waiting")
		assert.GreaterOrEqual(t, elapsed, 900*time.Millisecond, "should wait at least ~1s")
		assert.Less(t, elapsed, 1500*time.Millisecond, "should not wait much longer than 1s")
	})

	t.Run("uses code default windows", func(t *testing.T) {
		cleanupRedisKeys(t, rawRedis, "gmail", "global")

		limiter := createRedisLimiter(t, client, "gmail", nil)

		for i := 0; i < 40; i++ {
			err := limiter.Wait(ctx)
			assert.NoError(t, err, "request %d should succeed", i+1)
		}

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
func createRedisLimiter(t *testing.T, client redisclient.ClientInterface, namespace string, windows []config.Window) ratelimit.Limiter {
	t.Helper()
	return redislimit.NewLimiterWithWindows(client, nil, namespace, windows, nil)
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

func newRedisEnvWrapper(t *testing.T, vars map[string]string) *mocklibrary.OsWrapperMock {
	t.Helper()

	scriptPath := filepath.Join("..", "internal", "library", "redis", "script", "scripts", "rate_limit.lua")
	absPath, err := filepath.Abs(scriptPath)
	require.NoError(t, err, "failed to resolve rate_limit.lua path")

	body, err := os.ReadFile(absPath)
	require.NoError(t, err, "failed to read rate_limit.lua")

	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"RATE_LIMIT_SCRIPT_PATH": absPath,
	}).WithFile(absPath, string(body))

	for key, value := range vars {
		if value == "" {
			osw.WithEnvValue(key, value)
			continue
		}
		osw.WithEnv(map[string]string{key: value})
	}

	return osw
}

func envOrDefault(key, defaultValue string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultValue
}
