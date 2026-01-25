package domain

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents the user entity in the authentication domain
type User struct {
	ID              uint
	Name            string
	Email           string
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
