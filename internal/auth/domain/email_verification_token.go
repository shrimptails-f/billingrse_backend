package domain

import "time"

// EmailVerificationToken represents the email verification token entity
type EmailVerificationToken struct {
	ID         uint
	UserID     uint
	Token      string
	ExpiresAt  time.Time
	CreatedAt  time.Time
	ConsumedAt *time.Time
}
