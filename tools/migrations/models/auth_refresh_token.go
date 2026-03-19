package model

import "time"

// AuthRefreshToken represents the auth_refresh_tokens table.
type AuthRefreshToken struct {
	ID                uint      `gorm:"primaryKey;autoIncrement"`
	UserID            uint      `gorm:"not null;index"`
	TokenDigest       string    `gorm:"size:64;uniqueIndex:uni_auth_refresh_tokens_token_digest;not null"`
	ExpiresAt         time.Time `gorm:"not null"`
	LastUsedAt        *time.Time
	RevokedAt         *time.Time
	ReplacedByTokenID *uint `gorm:"index"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName specifies the table name for the AuthRefreshToken model.
func (AuthRefreshToken) TableName() string {
	return "auth_refresh_tokens"
}
