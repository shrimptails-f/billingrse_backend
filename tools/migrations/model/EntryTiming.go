package model

import (
	"time"
)

type EntryTiming struct {
	UserID    uint   `gorm:"primaryKey"`                  // ユーザーID
	EmailID   uint   `gorm:"primaryKey"`                  // ID
	StartDate string `gorm:"primaryKey;size:20;not null"` // 入場日（例: "2025/06/01"）
	CreatedAt time.Time
	UpdatedAt time.Time
	User      User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // ユーザー
}
