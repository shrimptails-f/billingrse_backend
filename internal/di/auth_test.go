package di

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type mockOsWrapper struct {
	envVars map[string]string
}

func (m *mockOsWrapper) GetEnv(key string) (string, error) {
	if m.envVars == nil {
		return "", fmt.Errorf("environment variable %s not set", key)
	}
	if value, ok := m.envVars[key]; ok && value != "" {
		return value, nil
	}
	return "", fmt.Errorf("environment variable %s not set", key)
}

func (m *mockOsWrapper) ReadFile(path string) (string, error) {
	return "", nil
}

func TestParseTokenTTL_Empty(t *testing.T) {
	osw := &mockOsWrapper{envVars: map[string]string{}}
	result := parseTokenTTL(osw)
	assert.Equal(t, 24*time.Hour, result, "Empty JWT_EXPIRES_IN should default to 24h")
}

func TestParseTokenTTL_DurationFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"30m", 30 * time.Minute},
		{"2h", 2 * time.Hour},
		{"90s", 90 * time.Second},
		{"24h", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			osw := &mockOsWrapper{envVars: map[string]string{"JWT_EXPIRES_IN": tt.input}}
			result := parseTokenTTL(osw)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseTokenTTL_NumericSeconds(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
	}{
		{"3600", 3600 * time.Second},
		{"7200", 7200 * time.Second},
		{"60", 60 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			osw := &mockOsWrapper{envVars: map[string]string{"JWT_EXPIRES_IN": tt.input}}
			result := parseTokenTTL(osw)
			assert.Equal(t, tt.expected, result, "Numeric value should be interpreted as seconds")
		})
	}
}

func TestParseTokenTTL_InvalidFormat(t *testing.T) {
	tests := []string{
		"invalid",
		"abc",
		"-100",
		"   ",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			osw := &mockOsWrapper{envVars: map[string]string{"JWT_EXPIRES_IN": input}}
			result := parseTokenTTL(osw)
			assert.Equal(t, 24*time.Hour, result, "Invalid value should fallback to 24h")
		})
	}
}

func TestParseTokenTTL_Whitespace(t *testing.T) {
	osw := &mockOsWrapper{envVars: map[string]string{"JWT_EXPIRES_IN": "  2h  "}}
	result := parseTokenTTL(osw)
	assert.Equal(t, 2*time.Hour, result, "Whitespace should be trimmed")
}
