package infrastructure

import (
	"business/internal/library/logger"
	mfapp "business/internal/mailfetch/application"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"fmt"
	"strings"
)

// DefaultMailFetcherFactory creates provider-specific fetchers for manualmailfetch.
type DefaultMailFetcherFactory struct {
	gmailBuilder *GmailSessionBuilder
	log          logger.Interface
}

// NewDefaultMailFetcherFactory creates a default fetcher factory.
func NewDefaultMailFetcherFactory(gmailBuilder *GmailSessionBuilder, log logger.Interface) *DefaultMailFetcherFactory {
	if log == nil {
		log = logger.NewNop()
	}
	return &DefaultMailFetcherFactory{
		gmailBuilder: gmailBuilder,
		log:          log.With(logger.Component("manual_mail_fetch_factory")),
	}
}

// Create chooses a fetcher implementation for the given provider.
func (f *DefaultMailFetcherFactory) Create(ctx context.Context, conn mfdomain.ConnectionRef) (mfapp.MailFetcher, error) {
	_ = ctx
	switch strings.ToLower(strings.TrimSpace(conn.Provider)) {
	case "gmail":
		return NewGmailMailFetcherAdapter(conn, f.gmailBuilder, f.log), nil
	default:
		return nil, fmt.Errorf("%w: %s", mfdomain.ErrProviderUnsupported, conn.Provider)
	}
}
