package model

import "time"

// OAuthPendingState represents the oauth_pending_states table for storing temporary OAuth states.
type OAuthPendingState struct {
	ID         uint       `gorm:"primaryKey;autoIncrement"`
	UserID     uint       `gorm:"not null;index"`
	State      string     `gorm:"type:varchar(255);not null;uniqueIndex"`
	ExpiresAt  time.Time  `gorm:"not null"`
	ConsumedAt *time.Time `gorm:"default:null"`
	CreatedAt  time.Time  `gorm:"not null"`
}

// TableName specifies the table name for the OAuthPendingState model.
func (OAuthPendingState) TableName() string {
	return "oauth_pending_states"
}
