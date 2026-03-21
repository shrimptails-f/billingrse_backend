package infrastructure

import (
	"business/internal/library/gmailService"
	"business/internal/library/logger"
	"context"
	"fmt"

	"golang.org/x/oauth2"
)

// GmailProfileFetcher fetches the Gmail email address from the API.
type GmailProfileFetcher struct {
	gmailSvc gmailService.ClientInterface
	log      logger.Interface
}

// NewGmailProfileFetcher creates a new GmailProfileFetcher.
func NewGmailProfileFetcher(gmailSvc gmailService.ClientInterface, log logger.Interface) *GmailProfileFetcher {
	if log == nil {
		log = logger.NewNop()
	}
	return &GmailProfileFetcher{
		gmailSvc: gmailSvc,
		log:      log.With(logger.Component("gmail_profile_fetcher")),
	}
}

// GetEmailAddress fetches the authenticated user's Gmail address.
func (f *GmailProfileFetcher) GetEmailAddress(ctx context.Context, token *oauth2.Token, cfg *oauth2.Config) (string, error) {
	reqLog := f.log
	if l, err := f.log.WithContext(ctx); err == nil {
		reqLog = l
	}

	tokenSource := cfg.TokenSource(ctx, token)
	svc, err := f.gmailSvc.CreateServiceWithTokenSource(ctx, tokenSource)
	if err != nil {
		reqLog.Error("external_api_failed",
			logger.String("provider", "gmail"),
			logger.String("operation", "new_service"),
			logger.Err(err),
		)
		return "", fmt.Errorf("failed to create gmail service: %w", err)
	}

	profile, err := svc.Users.GetProfile("me").Context(ctx).Do()
	if err != nil {
		reqLog.Error("external_api_failed",
			logger.String("provider", "gmail"),
			logger.String("operation", "get_profile"),
			logger.Err(err),
		)
		return "", fmt.Errorf("failed to fetch gmail profile: %w", err)
	}

	reqLog.Info("external_api_succeeded",
		logger.String("provider", "gmail"),
		logger.String("operation", "get_profile"),
	)

	return profile.EmailAddress, nil
}
