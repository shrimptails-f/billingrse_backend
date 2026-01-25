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

// User represents a user in the system.
type User struct {
	ID              uint
	Name            UserName
	Email           EmailAddress
	PasswordHash    string
	EmailVerified   bool
	EmailVerifiedAt *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// VerifyPassword verifies the provided password against the stored password hash
// using bcrypt. Returns true if the password matches, false otherwise.
func (u *User) VerifyPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}
