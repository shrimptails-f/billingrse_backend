package script

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockOsWrapper is a simple mock for testing script loading.
type mockOsWrapper struct {
	files map[string]string
	env   map[string]string
}

func newMockOsWrapper(files, env map[string]string) *mockOsWrapper {
	return &mockOsWrapper{
		files: files,
		env:   env,
	}
}

func (m *mockOsWrapper) ReadFile(path string) (string, error) {
	if content, ok := m.files[path]; ok {
		return content, nil
	}
	return "", os.ErrNotExist
}

func (m *mockOsWrapper) GetEnv(key string) (string, error) {
	if v, ok := m.env[key]; ok && v != "" {
		return v, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

func TestNew(t *testing.T) {
	t.Parallel()

	path := "/tmp/test.lua"
	scriptBody := "return 1"
	expectedHash := sha1.New()
	expectedHash.Write([]byte(scriptBody))
	expectedSHA := hex.EncodeToString(expectedHash.Sum(nil))

	osw := newMockOsWrapper(map[string]string{path: scriptBody}, nil)
	scr, err := New(osw, "test_script", path)
	require.NoError(t, err)

	assert.Equal(t, "test_script", scr.Name)
	assert.Equal(t, scriptBody, scr.Body)
	assert.Equal(t, expectedSHA, scr.SHA)
}

func TestNew_WithEnvOverride(t *testing.T) {
	t.Parallel()

	path := "/tmp/test_env.lua"
	customSHA := "custom_sha_from_env"
	body := "return 2"
	osw := newMockOsWrapper(
		map[string]string{path: body},
		map[string]string{"SCRIPT_SHA_test_env": customSHA},
	)

	scr, err := New(osw, "test_env", path)
	require.NoError(t, err)

	assert.Equal(t, "test_env", scr.Name)
	assert.Equal(t, customSHA, scr.SHA)
}

func TestNew_FileNotFound(t *testing.T) {
	t.Parallel()

	osw := newMockOsWrapper(
		map[string]string{},
		nil,
	)

	_, err := New(osw, "test", "/nonexistent/file.lua")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read script file")
}

func TestNew_UsesEnvPathWhenEmpty(t *testing.T) {
	t.Parallel()

	scriptBody := "-- rate limit script"
	customPath := "/custom/rate_limit.lua"

	osw := newMockOsWrapper(
		map[string]string{customPath: scriptBody},
		map[string]string{"RATE_LIMIT_SCRIPT_PATH": customPath},
	)

	scr, err := New(osw, "rate_limit", "")

	require.NoError(t, err)
	assert.Equal(t, "rate_limit", scr.Name)
	assert.Equal(t, scriptBody, scr.Body)
}

func TestNew_RateLimitPathMissing(t *testing.T) {
	t.Parallel()

	osw := newMockOsWrapper(
		map[string]string{},
		map[string]string{},
	)

	_, err := New(osw, "rate_limit", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve path")
}

func TestNew_RealFileWithEnvPath(t *testing.T) {
	// This test uses the actual rate_limit.lua file
	scriptPath := filepath.Join("scripts", "rate_limit.lua")
	absPath, err := filepath.Abs(scriptPath)
	require.NoError(t, err)

	// Verify the file exists
	_, err = os.Stat(absPath)
	require.NoError(t, err, "rate_limit.lua not found at %s", absPath)

	require.NoError(t, os.Setenv("RATE_LIMIT_SCRIPT_PATH", absPath))
	t.Cleanup(func() { _ = os.Unsetenv("RATE_LIMIT_SCRIPT_PATH") })

	osw := &realOsWrapper{}

	scr, err := New(osw, "rate_limit", "")
	require.NoError(t, err)
	assert.Equal(t, "rate_limit", scr.Name)
	assert.NotEmpty(t, scr.Body)
	assert.Contains(t, scr.Body, "Sliding window rate limit script")
	assert.NotEmpty(t, scr.SHA)
}

// realOsWrapper wraps the real file system for integration-style tests.
type realOsWrapper struct{}

func (r *realOsWrapper) ReadFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (r *realOsWrapper) GetEnv(key string) (string, error) {
	if value := os.Getenv(key); value != "" {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}
