package model

import "time"

// Agent represents the agents table for storing encrypted agent tokens
type Agent struct {
	ID                 uint       `gorm:"primaryKey;autoIncrement"`
	UserID             uint       `gorm:"not null;uniqueIndex:idx_agents_user_type"`
	Type               string     `gorm:"size:50;not null;uniqueIndex:idx_agents_user_type"`
	KeyVersion         int16      `gorm:"not null;default:1"`
	Token              []byte     `gorm:"type:blob;not null"`
	TokenDigest        []byte     `gorm:"type:binary(32);not null;index"`
	RefreshToken       []byte     `gorm:"type:blob"`
	RefreshTokenDigest []byte     `gorm:"type:binary(32);index"`
	ExpiresAt          *time.Time `gorm:"default:null"`
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// TableName specifies the table name for the Agent model
func (Agent) TableName() string {
	return "agents"
}
