package model

import "time"

// EmailVerificationToken represents the email_verification_tokens table
type EmailVerificationToken struct {
	ID         uint      `gorm:"primaryKey;autoIncrement"`
	UserID     uint      `gorm:"not null;uniqueIndex:idx_user_id_unique"`
	Token      string    `gorm:"size:36;unique;not null;index:idx_token"`
	ExpiresAt  time.Time `gorm:"not null"`
	CreatedAt  time.Time `gorm:"not null"`
	ConsumedAt *time.Time
}

// TableName specifies the table name for the EmailVerificationToken model
func (EmailVerificationToken) TableName() string {
	return "email_verification_tokens"
}
