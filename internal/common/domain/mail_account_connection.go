package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	// ErrMailAccountConnectionUserIDEmpty is returned when the user ID is empty.
	ErrMailAccountConnectionUserIDEmpty = errors.New("mail account connection user id is empty")
	// ErrMailAccountConnectionAccessTokenEmpty is returned when the access token is empty.
	ErrMailAccountConnectionAccessTokenEmpty = errors.New("mail account connection access token is empty")
	// ErrMailAccountConnectionRefreshTokenEmpty is returned when the refresh token is empty.
	ErrMailAccountConnectionRefreshTokenEmpty = errors.New("mail account connection refresh token is empty")
	// ErrOAuthStateExpiresAtEmpty is returned when the OAuth state expiry is missing.
	ErrOAuthStateExpiresAtEmpty = errors.New("oauth state expires at is empty")
)

// MailAccountConnection represents a user's mail account connection.
type MailAccountConnection struct {
	ID                  uint
	UserID              uint
	AccessToken         string
	RefreshToken        string
	OAuthStateExpiresAt *time.Time
}

// Validate enforces invariants for MailAccountConnection.
func (c MailAccountConnection) Validate() error {
	if c.UserID == 0 {
		return ErrMailAccountConnectionUserIDEmpty
	}
	if strings.TrimSpace(c.AccessToken) == "" {
		return ErrMailAccountConnectionAccessTokenEmpty
	}
	if strings.TrimSpace(c.RefreshToken) == "" {
		return ErrMailAccountConnectionRefreshTokenEmpty
	}
	if c.OAuthStateExpiresAt != nil && c.OAuthStateExpiresAt.IsZero() {
		return ErrOAuthStateExpiresAtEmpty
	}
	return nil
}

// IsActive reports whether the authorization state is still valid.
func (c MailAccountConnection) IsActive() bool {
	if c.OAuthStateExpiresAt == nil {
		return false
	}
	return time.Now().Add(OAuthStateExpirySafetyOffset).Before(*c.OAuthStateExpiresAt)
}
