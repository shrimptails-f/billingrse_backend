package ratelimit

import (
	"business/internal/library/logger"
	"business/internal/library/oswrapper"
	redislimit "business/internal/library/ratelimit/limiter_redis"
	redisclient "business/internal/library/redis"
	"business/internal/library/timewrapper"
)

// Provider holds limiter instances for Gmail and OpenAI.
type Provider struct {
	gmailLimiter  Limiter
	openaiLimiter Limiter
}

// GetGmailLimiter returns the Gmail limiter instance.
func (p *Provider) GetGmailLimiter() Limiter {
	return p.gmailLimiter
}

// GetOpenAILimiter returns the OpenAI limiter instance.
func (p *Provider) GetOpenAILimiter() Limiter {
	return p.openaiLimiter
}

// NewProviderFromEnv constructs a Provider by reading Redis configuration from the environment.
func NewProviderFromEnv(osw oswrapper.OsWapperInterface, log logger.Interface) (*Provider, error) {
	if osw == nil {
		osw = oswrapper.New(nil)
	}
	if log == nil {
		log = logger.NewNop()
	}
	client, err := redisclient.New(redisclient.Config{}, osw, log)
	if err != nil {
		return nil, err
	}
	clock := timewrapper.NewClock()
	return NewProvider(client, clock, osw, log), nil
}

// NewProvider creates a new Provider with injected dependencies.
func NewProvider(
	client redisclient.ClientInterface,
	clock timewrapper.ClockInterface,
	osw oswrapper.OsWapperInterface,
	log logger.Interface,
) *Provider {
	if osw == nil {
		osw = oswrapper.New(nil)
	}
	if clock == nil {
		clock = timewrapper.NewClock()
	}
	if log == nil {
		log = logger.NewNop()
	}

	gmailLimiter := redislimit.NewLimiter(client, clock, "gmail", log)
	openaiLimiter := redislimit.NewLimiter(client, clock, "openai", log)

	return &Provider{
		gmailLimiter:  gmailLimiter,
		openaiLimiter: openaiLimiter,
	}
}
