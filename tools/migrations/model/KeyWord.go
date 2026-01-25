package model

import (
	"time"
)

// KeyWord（表記ゆれ）
type KeyWord struct {
	ID        uint   `gorm:"primaryKey;autoIncrement"`
	Word      string `gorm:"size:255;not null;unique"`
	CreatedAt time.Time
	UpdatedAt time.Time
}
