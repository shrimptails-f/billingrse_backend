package model

import "time"

// EmailCredential represents the email_credentials table for storing encrypted email OAuth tokens
type EmailCredential struct {
	ID                  uint       `gorm:"primaryKey;autoIncrement"`
	UserID              uint       `gorm:"not null;uniqueIndex:idx_email_credentials_user_type"`
	Type                string     `gorm:"size:50;not null;uniqueIndex:idx_email_credentials_user_type"`
	KeyVersion          int16      `gorm:"not null;default:1"`
	AccessToken         string     `gorm:"type:text;not null"`
	AccessTokenDigest   string     `gorm:"type:text;not null"`
	RefreshToken        string     `gorm:"type:text;not null"`
	RefreshTokenDigest  string     `gorm:"type:text;not null"`
	TokenExpiry         *time.Time `gorm:"default:null"`
	OAuthState          *string    `gorm:"type:varchar(255);default:null;index"`
	OAuthStateExpiresAt *time.Time `gorm:"default:null"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

// TableName specifies the table name for the EmailCredential model
func (EmailCredential) TableName() string {
	return "email_credentials"
}
