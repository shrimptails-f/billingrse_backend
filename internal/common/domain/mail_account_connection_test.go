package domain

import (
	"errors"
	"testing"
	"time"
)

func TestMailAccountConnectionValidate(t *testing.T) {
	t.Parallel()

	t.Run("returns error when user id is empty", func(t *testing.T) {
		t.Parallel()

		conn := MailAccountConnection{}
		if err := conn.Validate(); !errors.Is(err, ErrMailAccountConnectionUserIDEmpty) {
			t.Fatalf("expected ErrMailAccountConnectionUserIDEmpty, got %v", err)
		}
	})

	t.Run("returns error when oauth state expiry is zero time", func(t *testing.T) {
		t.Parallel()

		zero := time.Time{}
		conn := MailAccountConnection{UserID: 1, OAuthStateExpiresAt: &zero}
		if err := conn.Validate(); !errors.Is(err, ErrOAuthStateExpiresAtEmpty) {
			t.Fatalf("expected ErrOAuthStateExpiresAtEmpty, got %v", err)
		}
	})

	t.Run("succeeds when valid", func(t *testing.T) {
		t.Parallel()

		expiresAt := time.Now().Add(1 * time.Hour)
		conn := MailAccountConnection{UserID: 1, OAuthStateExpiresAt: &expiresAt}
		if err := conn.Validate(); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	})
}

func TestMailAccountConnectionIsActive(t *testing.T) {
	t.Parallel()

	t.Run("returns false when expires at is nil", func(t *testing.T) {
		t.Parallel()

		conn := MailAccountConnection{UserID: 1}
		if conn.IsActive() {
			t.Fatalf("expected IsActive to be false")
		}
	})

	t.Run("returns true when expiry is sufficiently in the future", func(t *testing.T) {
		t.Parallel()

		expiresAt := time.Now().Add(OAuthStateExpirySafetyOffset + time.Second)
		conn := MailAccountConnection{UserID: 1, OAuthStateExpiresAt: &expiresAt}
		if !conn.IsActive() {
			t.Fatalf("expected IsActive to be true")
		}
	})

	t.Run("returns false when expiry is within safety offset", func(t *testing.T) {
		t.Parallel()

		expiresAt := time.Now().Add(OAuthStateExpirySafetyOffset - time.Second)
		conn := MailAccountConnection{UserID: 1, OAuthStateExpiresAt: &expiresAt}
		if conn.IsActive() {
			t.Fatalf("expected IsActive to be false")
		}
	})
}
