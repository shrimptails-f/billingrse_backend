package model

import "time"

// User represents the user table for authentication
type User struct {
	ID              uint       `gorm:"primaryKey;autoIncrement"`
	Name            string     `gorm:"size:255;not null"`
	Email           string     `gorm:"size:255;unique;not null"`
	Password        string     `gorm:"size:255;not null"`
	EmailVerified   bool       `gorm:"not null;default:false"`
	EmailVerifiedAt *time.Time `gorm:"default:null"`
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TableName specifies the table name for the User model
func (User) TableName() string {
	return "users"
}
