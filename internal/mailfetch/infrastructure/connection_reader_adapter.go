package infrastructure

import (
	"business/internal/library/logger"
	macdomain "business/internal/mailaccountconnection/domain"
	mfdomain "business/internal/mailfetch/domain"
	"context"
	"errors"
	"fmt"
	"strings"
)

type connectionCredentialReader interface {
	FindCredentialByIDAndUser(ctx context.Context, credentialID, userID uint) (macdomain.EmailCredential, error)
}

// MailAccountConnectionReaderAdapter resolves fetchable connection metadata from email_credentials.
type MailAccountConnectionReaderAdapter struct {
	repo connectionCredentialReader
	log  logger.Interface
}

// NewMailAccountConnectionReaderAdapter creates a new connection reader adapter.
func NewMailAccountConnectionReaderAdapter(repo connectionCredentialReader, log logger.Interface) *MailAccountConnectionReaderAdapter {
	if log == nil {
		log = logger.NewNop()
	}
	return &MailAccountConnectionReaderAdapter{
		repo: repo,
		log:  log.With(logger.Component("manual_mail_fetch_connection_reader")),
	}
}

// FindUsableConnection checks ownership and returns the provider/account metadata needed for fetching.
func (a *MailAccountConnectionReaderAdapter) FindUsableConnection(ctx context.Context, userID, connectionID uint) (mfdomain.ConnectionRef, error) {
	credential, err := a.repo.FindCredentialByIDAndUser(ctx, connectionID, userID)
	if err != nil {
		if errors.Is(err, macdomain.ErrCredentialNotFound) {
			return mfdomain.ConnectionRef{}, mfdomain.ErrConnectionNotFound
		}
		return mfdomain.ConnectionRef{}, fmt.Errorf("failed to resolve mail account connection: %w", err)
	}

	provider := strings.ToLower(strings.TrimSpace(credential.Type))
	accountIdentifier := strings.ToLower(strings.TrimSpace(credential.GmailAddress))
	if provider == "" || accountIdentifier == "" ||
		strings.TrimSpace(credential.AccessToken) == "" ||
		strings.TrimSpace(credential.RefreshToken) == "" {
		return mfdomain.ConnectionRef{}, mfdomain.ErrConnectionUnavailable
	}

	return mfdomain.ConnectionRef{
		ConnectionID:      credential.ID,
		UserID:            credential.UserID,
		Provider:          provider,
		AccountIdentifier: accountIdentifier,
	}, nil
}
