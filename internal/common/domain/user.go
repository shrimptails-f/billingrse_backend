package domain

import (
	"errors"
	"net/mail"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

var (
	// ErrUserNameEmpty is returned when the user name is empty.
	ErrUserNameEmpty = errors.New("user name is empty")
	// ErrEmailAddressEmpty is returned when the email address is empty.
	ErrEmailAddressEmpty = errors.New("email address is empty")
	// ErrEmailAddressInvalid is returned when the email address format is invalid.
	ErrEmailAddressInvalid = errors.New("email address is invalid")
)

// UserName represents a user's display name.
type UserName string

// NewUserName creates a UserName from a raw string.
func NewUserName(value string) (UserName, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ErrUserNameEmpty
	}
	return UserName(trimmed), nil
}

// String returns the raw string value.
func (n UserName) String() string {
	return string(n)
}

// EmailAddress represents a user's email address.
type EmailAddress string

// NewEmailAddress creates an EmailAddress from a raw string.
func NewEmailAddress(value string) (EmailAddress, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", ErrEmailAddressEmpty
	}

	addr, err := mail.ParseAddress(trimmed)
	if err != nil || addr.Address != trimmed {
		return "", ErrEmailAddressInvalid
	}

	return EmailAddress(trimmed), nil
}

// String returns the raw string value.
func (e EmailAddress) String() string {
	return string(e)
}

// PasswordHash represents a hashed password value.
type PasswordHash string

// NewPasswordHashFromPlaintext hashes a plaintext password using bcrypt.
func NewPasswordHashFromPlaintext(password string) (PasswordHash, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return PasswordHash(hashedPassword), nil
}

// NewPasswordHashFromHash wraps an existing password hash string.
func NewPasswordHashFromHash(hash string) PasswordHash {
	return PasswordHash(hash)
}

// String returns the raw string value.
func (h PasswordHash) String() string {
	return string(h)
}

// Verify compares a plaintext password against the stored hash.
func (h PasswordHash) Verify(password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(password)) == nil
}

// User represents a user in the system.
type User struct {
	ID              uint
	Name            UserName
	Email           EmailAddress
	PasswordHash    PasswordHash
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// IsEmailVerified reports whether the user has completed email verification.
func (u User) IsEmailVerified() bool {
	return u.EmailVerifiedAt != nil
}

// VerifyPassword verifies the provided password against the stored password hash
// using bcrypt. Returns true if the password matches, false otherwise.
func (u *User) VerifyPassword(password string) bool {
	return u.PasswordHash.Verify(password)
}
