package domain

import (
	"errors"
	"testing"
	"time"
)

func TestNewUserName(t *testing.T) {
	t.Parallel()

	t.Run("trims and accepts non-empty name", func(t *testing.T) {
		t.Parallel()

		name, err := NewUserName("  Alice  ")
		if err != nil {
			t.Fatalf("NewUserName returned error: %v", err)
		}
		if name.String() != "Alice" {
			t.Fatalf("want Alice, got %s", name.String())
		}
	})

	t.Run("rejects empty name", func(t *testing.T) {
		t.Parallel()

		if _, err := NewUserName("   "); !errors.Is(err, ErrUserNameEmpty) {
			t.Fatalf("expected ErrUserNameEmpty, got %v", err)
		}
	})
}

func TestNewEmailAddress(t *testing.T) {
	t.Parallel()

	t.Run("trims and accepts valid email", func(t *testing.T) {
		t.Parallel()

		email, err := NewEmailAddress("  user@example.com ")
		if err != nil {
			t.Fatalf("NewEmailAddress returned error: %v", err)
		}
		if email.String() != "user@example.com" {
			t.Fatalf("want user@example.com, got %s", email.String())
		}
	})

	t.Run("rejects empty email", func(t *testing.T) {
		t.Parallel()

		if _, err := NewEmailAddress("   "); !errors.Is(err, ErrEmailAddressEmpty) {
			t.Fatalf("expected ErrEmailAddressEmpty, got %v", err)
		}
	})

	t.Run("rejects invalid email", func(t *testing.T) {
		t.Parallel()

		if _, err := NewEmailAddress("not-an-email"); !errors.Is(err, ErrEmailAddressInvalid) {
			t.Fatalf("expected ErrEmailAddressInvalid, got %v", err)
		}
	})
}

func TestPasswordHash(t *testing.T) {
	t.Parallel()

	t.Run("hash and verify password", func(t *testing.T) {
		t.Parallel()

		hash, err := NewPasswordHashFromPlaintext("password123")
		if err != nil {
			t.Fatalf("NewPasswordHashFromPlaintext returned error: %v", err)
		}
		if !hash.Verify("password123") {
			t.Fatalf("expected password to verify")
		}
		if hash.Verify("wrong-password") {
			t.Fatalf("expected password verification to fail")
		}
	})

	t.Run("wrap existing hash", func(t *testing.T) {
		t.Parallel()

		hash, err := NewPasswordHashFromPlaintext("secret")
		if err != nil {
			t.Fatalf("NewPasswordHashFromPlaintext returned error: %v", err)
		}
		wrapped := NewPasswordHashFromHash(hash.String())
		if wrapped.String() != hash.String() {
			t.Fatalf("expected wrapped hash to match original")
		}
	})
}

func TestUserVerification(t *testing.T) {
	t.Parallel()

	t.Run("email verified when timestamp set", func(t *testing.T) {
		t.Parallel()

		now := time.Now()
		user := User{EmailVerifiedAt: &now}
		if !user.IsEmailVerified() {
			t.Fatalf("expected email to be verified")
		}
	})

	t.Run("email not verified when timestamp is nil", func(t *testing.T) {
		t.Parallel()

		user := User{}
		if user.IsEmailVerified() {
			t.Fatalf("expected email to be unverified")
		}
	})
}

func TestUserVerifyPassword(t *testing.T) {
	t.Parallel()

	hash, err := NewPasswordHashFromPlaintext("password123")
	if err != nil {
		t.Fatalf("NewPasswordHashFromPlaintext returned error: %v", err)
	}
	user := User{PasswordHash: hash}

	if !user.VerifyPassword("password123") {
		t.Fatalf("expected VerifyPassword to succeed")
	}
	if user.VerifyPassword("wrong-password") {
		t.Fatalf("expected VerifyPassword to fail")
	}
}
