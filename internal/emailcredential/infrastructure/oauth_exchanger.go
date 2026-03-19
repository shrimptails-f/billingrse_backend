package infrastructure

import (
	"context"

	"golang.org/x/oauth2"
)

// OAuthTokenExchanger exchanges an authorization code for an OAuth2 token.
type OAuthTokenExchanger struct{}

// NewOAuthTokenExchanger creates a new OAuthTokenExchanger.
func NewOAuthTokenExchanger() *OAuthTokenExchanger {
	return &OAuthTokenExchanger{}
}

// Exchange exchanges the authorization code for a token using the provided config.
func (e *OAuthTokenExchanger) Exchange(ctx context.Context, cfg *oauth2.Config, code string) (*oauth2.Token, error) {
	return cfg.Exchange(ctx, code)
}
