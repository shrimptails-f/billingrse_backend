package script

import (
	"crypto/sha1"
	"encoding/hex"
	"os"
	"path/filepath"
	"testing"

	mocklibrary "business/test/mock/library"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	path := "/tmp/test.lua"
	scriptBody := "return 1"
	expectedHash := sha1.New()
	expectedHash.Write([]byte(scriptBody))
	expectedSHA := hex.EncodeToString(expectedHash.Sum(nil))

	osw := mocklibrary.NewOsWrapperMock(nil).WithFile(path, scriptBody)
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
	osw := mocklibrary.NewOsWrapperMock(map[string]string{"SCRIPT_SHA_test_env": customSHA}).WithFile(path, body)

	scr, err := New(osw, "test_env", path)
	require.NoError(t, err)

	assert.Equal(t, "test_env", scr.Name)
	assert.Equal(t, customSHA, scr.SHA)
}

func TestNew_FileNotFound(t *testing.T) {
	t.Parallel()

	osw := mocklibrary.NewOsWrapperMock(nil).WithReadFileError(os.ErrNotExist)

	_, err := New(osw, "test", "/nonexistent/file.lua")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to read script file")
}

func TestNew_UsesEnvPathWhenEmpty(t *testing.T) {
	t.Parallel()

	scriptBody := "-- rate limit script"
	customPath := "/custom/rate_limit.lua"

	osw := mocklibrary.NewOsWrapperMock(map[string]string{"RATE_LIMIT_SCRIPT_PATH": customPath}).WithFile(customPath, scriptBody)

	scr, err := New(osw, "rate_limit", "")

	require.NoError(t, err)
	assert.Equal(t, "rate_limit", scr.Name)
	assert.Equal(t, scriptBody, scr.Body)
}

func TestNew_RateLimitPathMissing(t *testing.T) {
	t.Parallel()

	osw := mocklibrary.NewOsWrapperMock(nil)

	_, err := New(osw, "rate_limit", "")

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to resolve path")
}

func TestNew_RealFileWithEnvPath(t *testing.T) {
	scriptPath := filepath.Join("scripts", "rate_limit.lua")
	absPath, err := filepath.Abs(scriptPath)
	require.NoError(t, err)

	// Verify the file exists
	_, err = os.Stat(absPath)
	require.NoError(t, err, "rate_limit.lua not found at %s", absPath)

	body, err := os.ReadFile(absPath)
	require.NoError(t, err)

	osw := mocklibrary.NewOsWrapperMock(map[string]string{"RATE_LIMIT_SCRIPT_PATH": absPath}).WithFile(absPath, string(body))

	scr, err := New(osw, "rate_limit", "")
	require.NoError(t, err)
	assert.Equal(t, "rate_limit", scr.Name)
	assert.NotEmpty(t, scr.Body)
	assert.Contains(t, scr.Body, "Sliding window rate limit script")
	assert.NotEmpty(t, scr.SHA)
}
