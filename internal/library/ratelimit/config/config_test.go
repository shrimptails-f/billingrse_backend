package config

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseRPS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		namespace string
		envVars   map[string]string
		expected  int
	}{
		{
			name:      "use REDIS_RATE_LIMIT_RPS when set",
			namespace: "gmail",
			envVars: map[string]string{
				"REDIS_RATE_LIMIT_RPS": "20",
			},
			expected: 20,
		},
		{
			name:      "fallback to GMAIL_API_REQUESTS_PER_SECOND",
			namespace: "gmail",
			envVars: map[string]string{
				"GMAIL_API_REQUESTS_PER_SECOND": "15",
			},
			expected: 15,
		},
		{
			name:      "fallback to OPENAI_API_REQUESTS_PER_SECOND",
			namespace: "openai",
			envVars: map[string]string{
				"OPENAI_API_REQUESTS_PER_SECOND": "25",
			},
			expected: 25,
		},
		{
			name:      "use default when nothing is set",
			namespace: "gmail",
			envVars:   map[string]string{},
			expected:  defaultRPS,
		},
		{
			name:      "ignore invalid RPS value",
			namespace: "gmail",
			envVars: map[string]string{
				"REDIS_RATE_LIMIT_RPS": "invalid",
			},
			expected: defaultRPS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.envVars)

			result := BaseRPS(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestParseWindowConfig(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		config   string
		expected []Window
	}{
		{
			name:   "parse valid config",
			config: "1:10,10:50,60:300",
			expected: []Window{
				{SizeSeconds: 1, Limit: 10},
				{SizeSeconds: 10, Limit: 50},
				{SizeSeconds: 60, Limit: 300},
			},
		},
		{
			name:   "handle whitespace",
			config: " 1 : 10 , 10 : 50 ",
			expected: []Window{
				{SizeSeconds: 1, Limit: 10},
				{SizeSeconds: 10, Limit: 50},
			},
		},
		{
			name:     "skip invalid entries",
			config:   "1:10,invalid,60:300",
			expected: []Window{{SizeSeconds: 1, Limit: 10}, {SizeSeconds: 60, Limit: 300}},
		},
		{
			name:     "return empty for invalid format",
			config:   "invalid",
			expected: []Window{},
		},
		{
			name:     "ignore negative values",
			config:   "-1:10,10:-50",
			expected: []Window{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseWindowConfig(tt.config)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWindows(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		namespace string
		envVars   map[string]string
		expected  []Window
	}{
		{
			name:      "use REDIS_RATE_LIMIT_WINDOW_CONFIG when set",
			namespace: "gmail",
			envVars: map[string]string{
				"REDIS_RATE_LIMIT_WINDOW_CONFIG": "1:5,10:25,60:150",
			},
			expected: []Window{
				{SizeSeconds: 1, Limit: 5},
				{SizeSeconds: 10, Limit: 25},
				{SizeSeconds: 60, Limit: 150},
			},
		},
		{
			name:      "generate defaults based on RPS",
			namespace: "gmail",
			envVars: map[string]string{
				"GMAIL_API_REQUESTS_PER_SECOND": "20",
			},
			expected: []Window{
				{SizeSeconds: 1, Limit: 20},
				{SizeSeconds: 10, Limit: 100},
				{SizeSeconds: 60, Limit: 600},
			},
		},
		{
			name:      "use default RPS when nothing is set",
			namespace: "gmail",
			envVars:   map[string]string{},
			expected: []Window{
				{SizeSeconds: 1, Limit: defaultRPS},
				{SizeSeconds: 10, Limit: defaultRPS * 5},
				{SizeSeconds: 60, Limit: defaultRPS * 30},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			withEnv(t, tt.envVars)

			result := Windows(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}

var envTestMu sync.Mutex

func withEnv(t *testing.T, vars map[string]string) {
	t.Helper()
	// copy map to avoid accidental modification
	envs := make(map[string]string, len(vars))
	for k, v := range vars {
		envs[k] = v
	}
	envTestMu.Lock()
	SetEnvGetter(func(key string) string {
		if v, ok := envs[key]; ok {
			return v
		}
		return ""
	})
	t.Cleanup(func() {
		SetEnvGetter(nil)
		envTestMu.Unlock()
	})
}
