package domain

import cd "business/internal/common/domain"

// User represents the user entity in the authentication domain.
type User = cd.User

// UserName represents a user's display name.
type UserName = cd.UserName

// EmailAddress represents a user's email address.
type EmailAddress = cd.EmailAddress

// PasswordHash represents a hashed password value.
type PasswordHash = cd.PasswordHash

// ErrUserNameEmpty is returned when the user name is empty.
var ErrUserNameEmpty = cd.ErrUserNameEmpty

// ErrEmailAddressEmpty is returned when the email address is empty.
var ErrEmailAddressEmpty = cd.ErrEmailAddressEmpty

// ErrEmailAddressInvalid is returned when the email address format is invalid.
var ErrEmailAddressInvalid = cd.ErrEmailAddressInvalid

// NewUserName creates a UserName from a raw string.
func NewUserName(value string) (UserName, error) {
	return cd.NewUserName(value)
}

// NewEmailAddress creates an EmailAddress from a raw string.
func NewEmailAddress(value string) (EmailAddress, error) {
	return cd.NewEmailAddress(value)
}

// NewPasswordHashFromPlaintext hashes a plaintext password using bcrypt.
func NewPasswordHashFromPlaintext(password string) (PasswordHash, error) {
	return cd.NewPasswordHashFromPlaintext(password)
}

// NewPasswordHashFromHash wraps an existing password hash string.
func NewPasswordHashFromHash(hash string) PasswordHash {
	return cd.NewPasswordHashFromHash(hash)
}
