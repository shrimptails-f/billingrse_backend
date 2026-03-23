package infrastructure

import (
	cd "business/internal/common/domain"
	"business/internal/library/gmail"
	"business/internal/library/logger"
	macdomain "business/internal/mailaccountconnection/domain"
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/oauth2"
	gmailapi "google.golang.org/api/gmail/v1"
)

type gmailMessageClient interface {
	GetMessagesByLabelName(ctx context.Context, labelName string, startDate time.Time) ([]string, error)
	GetGmailDetail(ctx context.Context, id string) (cd.FetchedEmailDTO, error)
}

type credentialReader interface {
	FindCredentialByIDAndUser(ctx context.Context, credentialID, userID uint) (macdomain.EmailCredential, error)
}

type tokenDecryptor interface {
	DecryptFromString(ciphertext string) (string, error)
}

type oauthConfigProvider interface {
	GetGmailOAuthConfig(ctx context.Context) (*oauth2.Config, error)
}

type gmailServiceFactory interface {
	CreateServiceWithTokenSource(ctx context.Context, tokenSource oauth2.TokenSource) (*gmailapi.Service, error)
}

type gmailClientBinder interface {
	SetClient(svc *gmailapi.Service) *gmail.Client
}

// GmailSessionBuilder restores a Gmail session from a stored mail-account connection.
type GmailSessionBuilder struct {
	credentialReader credentialReader
	vault            tokenDecryptor
	oauthConfig      oauthConfigProvider
	gmailService     gmailServiceFactory
	gmailClient      gmailClientBinder
	log              logger.Interface
}

// NewGmailSessionBuilder creates a Gmail session builder.
func NewGmailSessionBuilder(
	credentialReader credentialReader,
	vault tokenDecryptor,
	oauthConfig oauthConfigProvider,
	gmailService gmailServiceFactory,
	gmailClient gmailClientBinder,
	log logger.Interface,
) *GmailSessionBuilder {
	if log == nil {
		log = logger.NewNop()
	}
	return &GmailSessionBuilder{
		credentialReader: credentialReader,
		vault:            vault,
		oauthConfig:      oauthConfig,
		gmailService:     gmailService,
		gmailClient:      gmailClient,
		log:              log.With(logger.Component("manual_mail_fetch_gmail_session_builder")),
	}
}

// Build restores a provider client for the requested connection.
func (b *GmailSessionBuilder) Build(ctx context.Context, connectionID, userID uint) (gmailMessageClient, error) {
	credential, err := b.credentialReader.FindCredentialByIDAndUser(ctx, connectionID, userID)
	if err != nil {
		if errors.Is(err, macdomain.ErrCredentialNotFound) {
			return nil, err
		}
		return nil, fmt.Errorf("failed to load credential: %w", err)
	}

	accessToken, err := b.vault.DecryptFromString(credential.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt access token: %w", err)
	}

	refreshToken, err := b.vault.DecryptFromString(credential.RefreshToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt refresh token: %w", err)
	}

	cfg, err := b.oauthConfig.GetGmailOAuthConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to load gmail oauth config: %w", err)
	}

	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		TokenType:    "Bearer",
	}
	if credential.TokenExpiry != nil {
		token.Expiry = *credential.TokenExpiry
	}

	service, err := b.gmailService.CreateServiceWithTokenSource(ctx, cfg.TokenSource(ctx, token))
	if err != nil {
		return nil, fmt.Errorf("failed to create gmail service: %w", err)
	}

	return b.gmailClient.SetClient(service), nil
}
