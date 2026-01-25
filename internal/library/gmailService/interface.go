package gmailService

import (
	"context"

	"golang.org/x/oauth2"
	"google.golang.org/api/gmail/v1"
)

type ClientInterface interface {
	CreateServiceWithToken(ctx context.Context, credentialsPath string, token *oauth2.Token) (*gmail.Service, error)
	CreateServiceWithTokenSource(ctx context.Context, tokenSource oauth2.TokenSource) (*gmail.Service, error)
}
