package config

import (
	"business/internal/common"
	"strconv"
	"strings"
)

// Window defines a sliding window configuration for rate limiting.
type Window struct {
	SizeSeconds int // Window size in seconds
	Limit       int // Maximum requests allowed in this window
}

// BaseRPS returns the base RPS for Redis rate limiting.
// The value is fixed in code rather than configured via environment variables.
func BaseRPS(namespace string) int {
	return common.DefaultRedisRateLimitRPS
}

// Windows returns the window configurations for Redis rate limiting.
// The value is fixed in code rather than configured via environment variables.
func Windows(namespace string) []Window {
	windows := ParseWindowConfig(common.DefaultRedisRateLimitWindowConfig)
	if len(windows) > 0 {
		return windows
	}

	rps := BaseRPS(namespace)
	return []Window{
		{SizeSeconds: 1, Limit: rps},
		{SizeSeconds: 10, Limit: rps * 5},
		{SizeSeconds: 60, Limit: rps * 30},
	}
}

// ParseWindowConfig parses window configuration string like "1:10,10:50,60:300".
func ParseWindowConfig(config string) []Window {
	parts := strings.Split(config, ",")
	windows := make([]Window, 0, len(parts))

	for _, part := range parts {
		pair := strings.Split(strings.TrimSpace(part), ":")
		if len(pair) != 2 {
			continue
		}

		sizeSeconds, err1 := strconv.Atoi(strings.TrimSpace(pair[0]))
		limit, err2 := strconv.Atoi(strings.TrimSpace(pair[1]))

		if err1 == nil && err2 == nil && sizeSeconds > 0 && limit > 0 {
			windows = append(windows, Window{
				SizeSeconds: sizeSeconds,
				Limit:       limit,
			})
		}
	}

	return windows
}
