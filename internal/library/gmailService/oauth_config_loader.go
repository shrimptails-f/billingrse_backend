package gmailService

import (
	"business/internal/library/oswrapper"
	"context"
	"fmt"
	"strings"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	gmail "google.golang.org/api/gmail/v1"
)

// OAuthConfigLoader loads Gmail OAuth configuration from environment settings.
type OAuthConfigLoader struct {
	osw oswrapper.OsWapperInterface
}

// NewOAuthConfigLoader creates a new OAuthConfigLoader.
func NewOAuthConfigLoader(osw oswrapper.OsWapperInterface) *OAuthConfigLoader {
	return &OAuthConfigLoader{osw: osw}
}

// GetGmailOAuthConfig resolves the Gmail OAuth configuration.
func (l *OAuthConfigLoader) GetGmailOAuthConfig(ctx context.Context) (*oauth2.Config, error) {
	redirectURLRaw, err := l.osw.GetEnv("EMAIL_GMAIL_REDIRECT_URL")
	if err != nil {
		return nil, fmt.Errorf("failed to read EMAIL_GMAIL_REDIRECT_URL: %w", err)
	}
	redirectURL := strings.TrimSpace(redirectURLRaw)
	if redirectURL == "" {
		return nil, fmt.Errorf("EMAIL_GMAIL_REDIRECT_URL environment variable is required")
	}

	if cfg, err := l.loadInlineConfig(redirectURL); err != nil {
		return nil, err
	} else if cfg != nil {
		return cfg, nil
	}
	return nil, fmt.Errorf("EMAIL_GMAIL_CLIENT_ID and EMAIL_GMAIL_CLIENT_SECRET are required")
}

func (l *OAuthConfigLoader) loadInlineConfig(redirectURL string) (*oauth2.Config, error) {
	clientID, errID := l.osw.GetEnv("EMAIL_GMAIL_CLIENT_ID")
	if errID != nil {
		return nil, fmt.Errorf("failed to read EMAIL_GMAIL_CLIENT_ID: %w", errID)
	}

	clientSecret, errSecret := l.osw.GetEnv("EMAIL_GMAIL_CLIENT_SECRET")
	if errSecret != nil {
		return nil, fmt.Errorf("failed to read EMAIL_GMAIL_CLIENT_SECRET: %w", errSecret)
	}

	id := strings.TrimSpace(clientID)
	secret := strings.TrimSpace(clientSecret)
	if id == "" || secret == "" {
		return nil, fmt.Errorf("EMAIL_GMAIL_CLIENT_ID or EMAIL_GMAIL_CLIENT_SECRET is empty")
	}

	return &oauth2.Config{
		ClientID:     id,
		ClientSecret: secret,
		Endpoint:     google.Endpoint,
		Scopes:       []string{gmail.GmailReadonlyScope},
		RedirectURL:  redirectURL,
	}, nil
}
