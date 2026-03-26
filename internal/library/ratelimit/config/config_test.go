package config

import (
	"business/internal/common"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseRPS(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name      string
		namespace string
		expected  int
	}{
		{name: "gmail uses code default", namespace: "gmail", expected: common.DefaultRedisRateLimitRPS},
		{name: "openai uses code default", namespace: "openai", expected: common.DefaultRedisRateLimitRPS},
		{name: "other namespace uses code default", namespace: "other", expected: common.DefaultRedisRateLimitRPS},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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
		expected  []Window
	}{
		{
			name:      "gmail uses code default windows",
			namespace: "gmail",
			expected: []Window{
				{SizeSeconds: 1, Limit: 40},
				{SizeSeconds: 10, Limit: 400},
				{SizeSeconds: 60, Limit: 2400},
			},
		},
		{
			name:      "openai uses code default windows",
			namespace: "openai",
			expected: []Window{
				{SizeSeconds: 1, Limit: 40},
				{SizeSeconds: 10, Limit: 400},
				{SizeSeconds: 60, Limit: 2400},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Windows(tt.namespace)
			assert.Equal(t, tt.expected, result)
		})
	}
}
