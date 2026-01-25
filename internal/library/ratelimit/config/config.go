package config

import (
	"os"
	"strconv"
	"strings"
	"sync"
)

const (
	gmailEnvKey  = "GMAIL_API_REQUESTS_PER_SECOND"
	openAIEnvKey = "OPENAI_API_REQUESTS_PER_SECOND"
	defaultRPS   = 10
)

var (
	envGetter   = os.Getenv
	envGetterMu sync.RWMutex
)

// SetEnvGetter overrides the function used to retrieve environment variables.
// Pass nil to reset to os.Getenv.
func SetEnvGetter(getter func(string) string) {
	envGetterMu.Lock()
	defer envGetterMu.Unlock()
	if getter == nil {
		envGetter = os.Getenv
		return
	}
	envGetter = getter
}

func readEnv(key string) string {
	envGetterMu.RLock()
	defer envGetterMu.RUnlock()
	return envGetter(key)
}

// Window defines a sliding window configuration for rate limiting.
type Window struct {
	SizeSeconds int // Window size in seconds
	Limit       int // Maximum requests allowed in this window
}

// BaseRPS returns the base RPS for Redis rate limiting.
// Reads from REDIS_RATE_LIMIT_RPS or falls back to GMAIL_API_REQUESTS_PER_SECOND / OPENAI_API_REQUESTS_PER_SECOND.
func BaseRPS(namespace string) int {
	if rpsStr := readEnv("REDIS_RATE_LIMIT_RPS"); rpsStr != "" {
		if rps, err := strconv.Atoi(rpsStr); err == nil && rps > 0 {
			return rps
		}
	}

	// Fallback to namespace-specific RPS
	var envKey string
	if namespace == "gmail" {
		envKey = gmailEnvKey
	} else if namespace == "openai" {
		envKey = openAIEnvKey
	}

	if envKey != "" {
		if rpsStr := readEnv(envKey); rpsStr != "" {
			if rps, err := strconv.Atoi(rpsStr); err == nil && rps > 0 {
				return rps
			}
		}
	}

	return defaultRPS
}

// Windows returns the window configurations for Redis rate limiting.
// Reads from REDIS_RATE_LIMIT_WINDOW_CONFIG (format: "1:10,10:50,60:300") or generates defaults.
func Windows(namespace string) []Window {
	if config := readEnv("REDIS_RATE_LIMIT_WINDOW_CONFIG"); config != "" {
		windows := ParseWindowConfig(config)
		if len(windows) > 0 {
			return windows
		}
	}

	// Default: generate windows based on RPS
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
