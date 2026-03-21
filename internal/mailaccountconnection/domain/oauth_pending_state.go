package domain

import "time"

// OAuthPendingState represents a temporary OAuth state for the authorize flow.
type OAuthPendingState struct {
	ID         uint
	UserID     uint
	State      string
	ExpiresAt  time.Time
	ConsumedAt *time.Time
	CreatedAt  time.Time
}
