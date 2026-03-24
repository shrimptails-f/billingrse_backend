package model

import "time"

// Email represents the emails table for raw fetched email metadata.
type Email struct {
	ID                uint      `gorm:"primaryKey;autoIncrement"`
	UserID            uint      `gorm:"not null;uniqueIndex:uni_emails_user_message"`
	Provider          string    `gorm:"size:50;not null"`
	AccountIdentifier string    `gorm:"size:255;not null"`
	ExternalMessageID string    `gorm:"size:255;not null;uniqueIndex:uni_emails_user_message"`
	Subject           string    `gorm:"type:text;not null"`
	FromRaw           string    `gorm:"column:from_raw;type:text;not null"`
	ToJSON            string    `gorm:"column:to_json;type:json;not null"`
	ReceivedAt        time.Time `gorm:"not null"`
	CreatedRunID      *string   `gorm:"column:created_run_id;size:36"`
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

// TableName specifies the table name for the Email model.
func (Email) TableName() string {
	return "emails"
}
