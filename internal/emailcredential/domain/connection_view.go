package domain

import "time"

// ConnectionView is the provider-agnostic read model returned by the list API.
type ConnectionView struct {
	ID                uint      `json:"id"`
	Provider          string    `json:"provider"`
	AccountIdentifier string    `json:"account_identifier"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
