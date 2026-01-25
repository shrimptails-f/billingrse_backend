//go:build integration
// +build integration

package redisclient

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"business/internal/library/oswrapper"
	"business/internal/library/redis/script"

	"github.com/alicebob/miniredis/v2"
	goredis "github.com/redis/go-redis/v9"
	maintnotifications "github.com/redis/go-redis/v9/maintnotifications"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient_BuildsURLFromEnv(t *testing.T) {
	t.Parallel()
	scriptPath := "/tmp/rate_limit_stub.lua"
	defaultEnv := map[string]string{
		"REDIS_HOST":             "localhost",
		"REDIS_PORT":             "6379",
		"REDIS_PASSWORD":         "",
		"REDIS_DB":               "0",
		"SCRIPT_SHA_RATE_LIMIT":  "test-sha",
		"RATE_LIMIT_SCRIPT_PATH": scriptPath,
	}

	tests := []struct {
		name             string
		envVars          map[string]string
		expectedAddr     string
		expectedPassword string
		expectedDB       int
	}{
		{
			name: "builds from components with password",
			envVars: map[string]string{
				"REDIS_HOST":     "testhost",
				"REDIS_PORT":     "6380",
				"REDIS_PASSWORD": "testpass",
				"REDIS_DB":       "2",
			},
			expectedAddr:     "testhost:6380",
			expectedPassword: "testpass",
			expectedDB:       2,
		},
		{
			name: "builds from components without password",
			envVars: map[string]string{
				"REDIS_HOST": "localhost",
				"REDIS_PORT": "6379",
				"REDIS_DB":   "0",
			},
			expectedAddr:     "localhost:6379",
			expectedPassword: "",
			expectedDB:       0,
		},
		{
			name:             "uses defaults when nothing set",
			envVars:          map[string]string{},
			expectedAddr:     "localhost:6379",
			expectedPassword: "",
			expectedDB:       0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mergedEnv := make(map[string]string, len(defaultEnv)+len(tt.envVars))
			for k, v := range defaultEnv {
				mergedEnv[k] = v
			}
			for k, v := range tt.envVars {
				mergedEnv[k] = v
			}

			osw := &stubOsWrapper{
				vars: mergedEnv,
				files: map[string]string{
					scriptPath: "-- rate limit script",
				},
			}

			clientInterface, err := New(Config{}, osw, nil)
			require.NoError(t, err)
			require.NotNil(t, clientInterface)

			client, ok := clientInterface.(*Client)
			require.True(t, ok, "expected concrete redis client")

			opts := client.client.Options()
			assert.Equal(t, tt.expectedAddr, opts.Addr)
			assert.Equal(t, tt.expectedPassword, opts.Password)
			assert.Equal(t, tt.expectedDB, opts.DB)
		})
	}
}

func TestNewClient_UsesProvidedURL(t *testing.T) {
	t.Parallel()
	scriptPath := "/tmp/rate_limit_stub.lua"
	osw := &stubOsWrapper{
		vars: map[string]string{
			"REDIS_HOST":             "redis",
			"REDIS_PORT":             "6379",
			"REDIS_PASSWORD":         "redis_local_password",
			"REDIS_DB":               "3",
			"SCRIPT_SHA_RATE_LIMIT":  "test-sha",
			"RATE_LIMIT_SCRIPT_PATH": scriptPath,
		},
		files: map[string]string{
			scriptPath: "-- rate limit script",
		},
	}

	clientInterface, err := New(Config{}, osw, nil)
	require.NoError(t, err)
	require.NotNil(t, clientInterface)

	client, ok := clientInterface.(*Client)
	require.True(t, ok)

	opts := client.client.Options()
	assert.Equal(t, "redis:6379", opts.Addr)
	assert.Equal(t, "redis_local_password", opts.Password)
	assert.Equal(t, 3, opts.DB)
}

func TestNewClient_InvalidURL(t *testing.T) {
	t.Parallel()
	_, err := New(Config{URL: "://bad"}, nil, nil)
	require.Error(t, err)
}

type stubOsWrapper struct {
	vars  map[string]string
	files map[string]string
}

func (s *stubOsWrapper) ReadFile(path string) (string, error) {
	if s.files == nil {
		return "", fmt.Errorf("file %s not configured", path)
	}
	if data, ok := s.files[path]; ok {
		return data, nil
	}
	return "", fmt.Errorf("file %s not configured", path)
}

func (s *stubOsWrapper) GetEnv(key string) (string, error) {
	if s.vars != nil {
		if v, ok := s.vars[key]; ok {
			return v, nil
		}
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

type envOverrideOsWrapper struct {
	base oswrapper.OsWapperInterface
	env  map[string]string
}

func newEnvOverrideOsWrapper(env map[string]string) oswrapper.OsWapperInterface {
	return &envOverrideOsWrapper{
		base: oswrapper.New(nil),
		env:  env,
	}
}

func (m *envOverrideOsWrapper) ReadFile(path string) (string, error) {
	return m.base.ReadFile(path)
}

func (m *envOverrideOsWrapper) GetEnv(key string) (string, error) {
	if v, ok := m.env[key]; ok && v != "" {
		return v, nil
	}
	return m.base.GetEnv(key)
}

// setupTestScriptFile creates a temporary rate_limit.lua script for testing and
// configures the environment to point to it.
func setupTestScriptFile(t *testing.T) string {
	t.Helper()

	// Use the actual rate_limit.lua from the project
	scriptPath := filepath.Join("..", "..", "..", "internal", "library", "redis", "script", "scripts", "rate_limit.lua")
	absPath, err := filepath.Abs(scriptPath)
	require.NoError(t, err, "failed to get absolute path for test script")

	// Verify the file exists
	_, err = os.Stat(absPath)
	require.NoError(t, err, "rate_limit.lua not found at %s", absPath)

	// Override the os wrapper to return the custom script path
	SetDefaultOsWrapper(newEnvOverrideOsWrapper(map[string]string{
		"RATE_LIMIT_SCRIPT_PATH": absPath,
	}))
	t.Cleanup(func() { SetDefaultOsWrapper(nil) })

	return absPath
}

func loadScriptFromBody(t *testing.T, name, body string) script.Script {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name+".lua")
	err := os.WriteFile(path, []byte(body), 0o600)
	require.NoError(t, err)

	scr, err := script.New(oswrapper.New(nil), name, path)
	require.NoError(t, err)
	return scr
}

func TestNewClient_DisablesMaintNotificationsByDefault(t *testing.T) {
	t.Parallel()
	setupTestScriptFile(t)

	clientInterface, err := New(Config{}, nil, nil)
	require.NoError(t, err)
	require.NotNil(t, clientInterface)

	client, ok := clientInterface.(*Client)
	require.True(t, ok, "expected concrete client implementation")

	cfg := client.client.Options().MaintNotificationsConfig
	require.NotNil(t, cfg)
	require.Equal(t, maintnotifications.ModeDisabled, cfg.Mode)
}

// TestEvalScript_Success tests that EvalScript executes successfully when script is loaded.
func TestEvalScript_Success(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	defer mr.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	client := &Client{
		client:      rdb,
		scriptCache: make(map[string]string),
	}

	scr := loadScriptFromBody(t, "test", "return {ARGV[1], ARGV[2]}")

	// Load the script first
	_, err := rdb.ScriptLoad(context.Background(), scr.Body).Result()
	require.NoError(t, err)

	// Execute via EvalScript
	result, err := client.EvalScript(context.Background(), scr, []string{}, "foo", "bar")
	require.NoError(t, err)

	resultSlice, ok := result.([]interface{})
	require.True(t, ok)
	assert.Equal(t, "foo", resultSlice[0])
	assert.Equal(t, "bar", resultSlice[1])
}

// TestEvalScript_NOSCRIPTRecovery tests that EvalScript auto-recovers from NOSCRIPT errors.
func TestEvalScript_NOSCRIPTRecovery(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	defer mr.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	client := &Client{
		client:      rdb,
		scriptCache: make(map[string]string),
	}

	scr := loadScriptFromBody(t, "test", "return {ARGV[1], ARGV[2]}")

	// Do NOT load the script beforehand - this will cause NOSCRIPT on first EvalSha
	// EvalScript should handle it by loading the script and retrying

	result, err := client.EvalScript(context.Background(), scr, []string{}, "hello", "world")
	require.NoError(t, err, "EvalScript should recover from NOSCRIPT automatically")

	resultSlice, ok := result.([]interface{})
	require.True(t, ok)
	assert.Equal(t, "hello", resultSlice[0])
	assert.Equal(t, "world", resultSlice[1])

	// Verify the script is now cached
	assert.Contains(t, client.scriptCache, "test")
}

// TestEvalScript_NOSCRIPTMultipleGoroutines tests concurrent NOSCRIPT recovery.
func TestEvalScript_NOSCRIPTMultipleGoroutines(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	defer mr.Close()

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	client := &Client{
		client:      rdb,
		scriptCache: make(map[string]string),
	}

	scr := loadScriptFromBody(t, "test_concurrent", "return {ARGV[1]}")

	// Launch multiple goroutines trying to execute the script concurrently
	const numGoroutines = 10
	done := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			_, err := client.EvalScript(context.Background(), scr, []string{}, id)
			done <- err
		}(i)
	}

	// All should succeed without error
	for i := 0; i < numGoroutines; i++ {
		err := <-done
		assert.NoError(t, err, "goroutine %d should succeed", i)
	}

	// Script should be loaded only once
	assert.Contains(t, client.scriptCache, "test_concurrent")
}

// TestEvalScript_OtherRedisError tests that non-NOSCRIPT errors are returned as-is.
func TestEvalScript_OtherRedisError(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	addr := mr.Addr() // Save address before closing
	mr.Close()        // Close immediately to cause connection error

	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	client := &Client{
		client:      rdb,
		scriptCache: make(map[string]string),
	}

	scr := loadScriptFromBody(t, "test", "return 1")

	_, err := client.EvalScript(context.Background(), scr, []string{}, "arg")
	assert.Error(t, err, "should return connection error")
	assert.NotContains(t, err.Error(), "NOSCRIPT", "error should not be NOSCRIPT")
}

// TestRunRateLimitScript_Success tests successful rate limit script execution.
func TestRunRateLimitScript_Success(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	defer mr.Close()

	// Mock script that returns allowed=1, windowSize=1, limit=10, current=1
	mockScript := "return {1, 1, 10, 1}"

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	sha, err := rdb.ScriptLoad(context.Background(), mockScript).Result()
	require.NoError(t, err)

	client := &Client{
		client: rdb,
		scriptCache: map[string]string{
			"rate_limit": sha,
		},
		rateLimitScript: loadScriptFromBody(t, "rate_limit", mockScript),
	}

	params := RateLimitParams{
		Namespace: "test",
		Bucket:    "global",
		Time:      time.Now(),
		Windows:   nil,
	}

	result, err := client.RunRateLimitScript(context.Background(), params)
	require.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 1, result.WindowSeconds)
	assert.Equal(t, 10, result.Limit)
	assert.Equal(t, 1, result.Current)
}

// TestRunRateLimitScript_ParsesResultCorrectly tests result parsing.
func TestRunRateLimitScript_ParsesResultCorrectly(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	defer mr.Close()

	// Load a mock script that returns specific values
	mockScript := "return {1, 60, 100, 50}"

	rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
	sha, err := rdb.ScriptLoad(context.Background(), mockScript).Result()
	require.NoError(t, err)

	client := &Client{
		client: rdb,
		scriptCache: map[string]string{
			"rate_limit": sha,
		},
		rateLimitScript: loadScriptFromBody(t, "rate_limit", mockScript),
	}

	params := RateLimitParams{
		Namespace: "test",
		Bucket:    "global",
		Time:      time.Now(),
		Windows:   nil,
	}

	result, err := client.RunRateLimitScript(context.Background(), params)
	require.NoError(t, err)

	assert.True(t, result.Allowed)
	assert.Equal(t, 60, result.WindowSeconds)
	assert.Equal(t, 100, result.Limit)
	assert.Equal(t, 50, result.Current)
}

// TestRunRateLimitScript_RedisError tests error handling.
func TestRunRateLimitScript_RedisError(t *testing.T) {
	t.Parallel()
	mr := miniredis.RunT(t)
	addr := mr.Addr()
	mr.Close() // Close to cause connection error

	rdb := goredis.NewClient(&goredis.Options{Addr: addr})
	client := &Client{
		client:          rdb,
		scriptCache:     make(map[string]string),
		rateLimitScript: loadScriptFromBody(t, "rate_limit", "return {1, 1, 10, 1}"),
	}

	params := RateLimitParams{
		Namespace: "test",
		Bucket:    "global",
		Time:      time.Now(),
		Windows:   nil,
	}

	_, err := client.RunRateLimitScript(context.Background(), params)
	require.Error(t, err)

	var redisErr *ErrRedisUnavailable
	assert.ErrorAs(t, err, &redisErr)
}

// TestRunRateLimitScript_UnexpectedResponseFormat tests handling of invalid response formats.
func TestRunRateLimitScript_UnexpectedResponseFormat(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name        string
		scriptBody  string
		expectedErr string
	}{
		{
			name:        "not a slice",
			scriptBody:  "return 'invalid'",
			expectedErr: "unexpected redis response format",
		},
		{
			name:        "slice too short",
			scriptBody:  "return {1, 2}",
			expectedErr: "unexpected redis response format",
		},
		{
			name:        "empty slice",
			scriptBody:  "return {}",
			expectedErr: "unexpected redis response format",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			mr := miniredis.RunT(t)
			defer mr.Close()

			rdb := goredis.NewClient(&goredis.Options{Addr: mr.Addr()})
			sha, err := rdb.ScriptLoad(context.Background(), tc.scriptBody).Result()
			require.NoError(t, err)

			client := &Client{
				client: rdb,
				scriptCache: map[string]string{
					"rate_limit": sha,
				},
				rateLimitScript: loadScriptFromBody(t, "rate_limit", tc.scriptBody),
			}

			params := RateLimitParams{
				Namespace: "test",
				Bucket:    "global",
				Time:      time.Now(),
				Windows:   nil,
			}

			_, err = client.RunRateLimitScript(context.Background(), params)
			require.Error(t, err)

			var redisErr *ErrRedisUnavailable
			assert.ErrorAs(t, err, &redisErr)
			assert.Contains(t, err.Error(), tc.expectedErr)
		})
	}
}

func TestParseInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    interface{}
		expected int64
	}{
		{"int64", int64(123), 123},
		{"int", int(456), 456},
		{"string", "789", 789},
		{"invalid string", "invalid", 0},
		{"nil", nil, 0},
		{"float64", float64(3.14), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseInt64(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}
