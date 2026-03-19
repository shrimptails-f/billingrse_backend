package domain

import "errors"

var (
	// ErrPendingStateNotFound is returned when no pending state matches the given state string.
	ErrPendingStateNotFound = errors.New("pending state not found")
	// ErrCredentialNotFound is returned when no credential matches the given criteria.
	ErrCredentialNotFound = errors.New("credential not found")
	// ErrOAuthStateMismatch is returned when the provided state does not match any pending state.
	ErrOAuthStateMismatch = errors.New("oauth state mismatch")
	// ErrOAuthStateExpired is returned when the pending state has expired.
	ErrOAuthStateExpired = errors.New("oauth state expired")
	// ErrOAuthExchangeFailed is returned when the Google token exchange fails.
	ErrOAuthExchangeFailed = errors.New("gmail oauth exchange failed")
	// ErrGmailProfileFetchFailed is returned when fetching the Gmail profile fails.
	ErrGmailProfileFetchFailed = errors.New("gmail profile fetch failed")
	// ErrRefreshTokenMissing is returned when a new connection has no refresh token.
	ErrRefreshTokenMissing = errors.New("refresh token missing for new connection")
	// ErrVaultEncryptFailed is returned when token encryption fails.
	ErrVaultEncryptFailed = errors.New("vault encrypt failed")
)
