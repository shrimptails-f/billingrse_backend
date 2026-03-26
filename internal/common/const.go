package common

import "time"

const (
	// GmailOAuthStateTTL controls how long a pending Gmail OAuth state remains valid.
	GmailOAuthStateTTL = 10 * time.Minute

	// OAuthStateExpirySafetyOffset shortens the effective validity window for safety.
	OAuthStateExpirySafetyOffset = 10 * time.Minute

	// DefaultRedisRateLimitRPS is used when no runtime override is provided.
	DefaultRedisRateLimitRPS = 20

	// DefaultRedisRateLimitWindowConfig is the tuned default Redis sliding-window config.
	// It is intentionally explicit instead of being derived from DefaultRedisRateLimitRPS.
	DefaultRedisRateLimitWindowConfig = "1:40,10:400,60:2400"
)
