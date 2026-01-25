package redislimit

import (
	"context"
	"errors"
	"testing"
	"time"

	"business/internal/library/logger"
	redisclient "business/internal/library/redis"
	"business/internal/library/redis/script"

	"github.com/stretchr/testify/assert"
)

func TestLimiterWait_AllowsImmediately(t *testing.T) {
	t.Parallel()
	client := &mockRedisClient{
		responses: []mockResponse{
			{result: redisclient.RateLimitResult{Allowed: true, WindowSeconds: 1, Limit: 10, Current: 1}},
		},
	}
	clock := &stubClock{now: time.Unix(0, 0)}
	limiter := NewLimiter(client, clock, "gmail", logger.NewNop())

	err := limiter.Wait(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, 1, client.rateLimitCalls)
}

func TestLimiterWait_WaitsUntilAllowed(t *testing.T) {
	t.Parallel()
	client := &mockRedisClient{
		responses: []mockResponse{
			{result: redisclient.RateLimitResult{Allowed: false, WindowSeconds: 1, Limit: 10, Current: 10}},
			{result: redisclient.RateLimitResult{Allowed: true, WindowSeconds: 1, Limit: 10, Current: 10}},
		},
	}

	afterCh := make(chan time.Time, 1)
	clock := &stubClock{
		now: time.Unix(0, 0),
		afterFunc: func(d time.Duration) <-chan time.Time {
			return afterCh
		},
	}
	limiter := NewLimiter(client, clock, "gmail", logger.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		err := limiter.Wait(ctx)
		assert.NoError(t, err)
		close(done)
	}()

	// Allow the wait to proceed
	afterCh <- time.Unix(1, 0)

	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("limiter.Wait did not complete after receiving After signal")
	}

	assert.Equal(t, 2, client.rateLimitCalls)
}

func TestLimiterWait_RedisError(t *testing.T) {
	t.Parallel()
	expectedErr := &redisclient.ErrRedisUnavailable{Err: errors.New("redis failure")}
	client := &mockRedisClient{
		responses: []mockResponse{
			{err: expectedErr},
		},
	}
	clock := &stubClock{now: time.Unix(0, 0)}
	limiter := NewLimiter(client, clock, "gmail", logger.NewNop())

	err := limiter.Wait(context.Background())
	assert.Error(t, err)
	assert.True(t, errors.Is(err, expectedErr))
}

func TestLimiterWait_ContextCancelledWhileWaiting(t *testing.T) {
	t.Parallel()
	client := &mockRedisClient{
		responses: []mockResponse{
			{result: redisclient.RateLimitResult{Allowed: false, WindowSeconds: 5, Limit: 10, Current: 10}},
		},
	}

	clock := &stubClock{
		now: time.Unix(0, 0),
		afterFunc: func(d time.Duration) <-chan time.Time {
			// never signal
			return make(chan time.Time)
		},
	}
	limiter := NewLimiter(client, clock, "gmail", logger.NewNop())

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := limiter.Wait(ctx)
	assert.ErrorIs(t, err, context.Canceled)
}

func TestLimiterWait_NilClient(t *testing.T) {
	t.Parallel()
	clock := &stubClock{now: time.Unix(0, 0)}

	// nil client
	limiter := NewLimiter(nil, clock, "gmail", logger.NewNop())
	err := limiter.Wait(context.Background())
	assert.Error(t, err)
	var redisErr *redisclient.ErrRedisUnavailable
	assert.True(t, errors.As(err, &redisErr), "expected ErrRedisUnavailable for nil client")
}

type mockResponse struct {
	result redisclient.RateLimitResult
	err    error
}

type mockRedisClient struct {
	responses      []mockResponse
	rateLimitCalls int
}

func (m *mockRedisClient) RunRateLimitScript(ctx context.Context, params redisclient.RateLimitParams) (redisclient.RateLimitResult, error) {
	if m.rateLimitCalls >= len(m.responses) {
		return redisclient.RateLimitResult{}, &redisclient.ErrRedisUnavailable{
			Err: errors.New("unexpected RunRateLimitScript call"),
		}
	}
	resp := m.responses[m.rateLimitCalls]
	m.rateLimitCalls++
	if resp.err != nil {
		return redisclient.RateLimitResult{}, resp.err
	}
	return resp.result, nil
}

func (m *mockRedisClient) EvalScript(ctx context.Context, scr script.Script, keys []string, args ...interface{}) (interface{}, error) {
	return nil, errors.New("EvalScript should not be called directly in limiter tests")
}

func (m *mockRedisClient) EvalSha(ctx context.Context, sha string, keys []string, args ...interface{}) (interface{}, error) {
	return nil, errors.New("EvalSha should not be called in limiter tests")
}

type stubClock struct {
	now       time.Time
	afterFunc func(time.Duration) <-chan time.Time
}

func (s *stubClock) Now() time.Time {
	if s.now.IsZero() {
		return time.Now()
	}
	return s.now
}

func (s *stubClock) After(d time.Duration) <-chan time.Time {
	if s.afterFunc != nil {
		return s.afterFunc(d)
	}
	ch := make(chan time.Time, 1)
	ch <- s.Now().Add(d)
	return ch
}
