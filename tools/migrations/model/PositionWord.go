package model

import (
	"time"
)

// PositionWord（ポジションの表記ゆれ）
type PositionWord struct {
	ID              uint   `gorm:"primaryKey"`        // 表記ID
	PositionGroupID uint   `gorm:"not null"`          // 紐づくポジショングループID
	Word            string `gorm:"size:100;not null"` // 表記（例: "Project Manager", "ＰＭ"）
	CreatedAt       time.Time
	UpdatedAt       time.Time
}
