package model

import (
	"time"
)

// WorkType（業務マスタ）
type WorkType struct {
	ID        uint      `gorm:"primaryKey"`               // 業務ID
	Name      string    `gorm:"unique;size:100;not null"` // 業務名（例: バックエンド実装）
	CreatedAt time.Time // 作成日時
	UpdatedAt time.Time // 更新日時
}
