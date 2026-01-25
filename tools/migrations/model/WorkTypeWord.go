package model

import (
	"time"
)

// WorkTypeWord（業務表記ゆれ）
type WorkTypeWord struct {
	ID              uint   `gorm:"primaryKey"`        // 表記ID
	WorkTypeGroupID uint   `gorm:"not null"`          // 紐づく業務グループID
	Word            string `gorm:"size:100;not null"` // 表記（例: "BE実装", "バックエンド構築"）
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
