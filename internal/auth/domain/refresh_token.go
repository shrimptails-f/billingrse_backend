package domain

import (
	"time"
)

// RefreshToken represents a refresh session stored in the database.
type RefreshToken struct {
	ID                uint
	UserID            uint
	Token             string
	TokenDigest       string
	ExpiresAt         time.Time
	CreatedAt         time.Time
	LastUsedAt        *time.Time
	RevokedAt         *time.Time
	ReplacedByTokenID *uint
}

// IsActive reports whether the token is still usable at the given time.
func (t RefreshToken) IsActive(now time.Time) bool {
	return t.RevokedAt == nil && now.Before(t.ExpiresAt)
}
