package model

import (
	"time"
)

// WorkTypeGroup（業務種別の正規化グループ）
type WorkTypeGroup struct {
	WorkTypeGroupID uint      `gorm:"primaryKey"`               // 業務グループID
	Name            string    `gorm:"unique;size:100;not null"` // 正規化された業務名（例: "バックエンド開発"）
	CreatedAt       time.Time // 作成日時
	UpdatedAt       time.Time // 更新日時

	Words []WorkTypeWord `gorm:"foreignKey:WorkTypeGroupID;references:WorkTypeGroupID"` // 表記ゆれ一覧
}
