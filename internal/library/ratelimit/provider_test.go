package ratelimit

import (
	"testing"

	redislimit "business/internal/library/ratelimit/limiter_redis"
	"business/internal/library/timewrapper"
	mocklibrary "business/test/mock/library"

	"github.com/stretchr/testify/assert"
)

func TestNewProvider_ReturnsLimiters(t *testing.T) {
	t.Parallel()
	scriptPath := "/tmp/rate_limit_stub.lua"
	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"SCRIPT_SHA_RATE_LIMIT":  "test-sha",
		"RATE_LIMIT_SCRIPT_PATH": scriptPath,
	}).WithFile(scriptPath, "-- rate limit script")

	provider := NewProvider(nil, timewrapper.NewClock(), osw, nil)
	assert.NotNil(t, provider)
	assert.IsType(t, &redislimit.Limiter{}, provider.GetGmailLimiter())
	assert.IsType(t, &redislimit.Limiter{}, provider.GetOpenAILimiter())
}

func TestNewProviderFromEnv_DoesNotNil(t *testing.T) {
	t.Parallel()
	scriptPath := "/tmp/rate_limit_stub.lua"
	osw := mocklibrary.NewOsWrapperMock(map[string]string{
		"REDIS_HOST":             "redis",
		"REDIS_PORT":             "6379",
		"REDIS_PASSWORD":         "redis_local_password",
		"REDIS_DB":               "0",
		"SCRIPT_SHA_RATE_LIMIT":  "test-sha",
		"RATE_LIMIT_SCRIPT_PATH": scriptPath,
	}).WithFile(scriptPath, "-- rate limit script")

	provider, err := NewProviderFromEnv(osw, nil)
	assert.NoError(t, err)
	assert.NotNil(t, provider)
	assert.NotNil(t, provider.GetGmailLimiter())
	assert.NotNil(t, provider.GetOpenAILimiter())
}
