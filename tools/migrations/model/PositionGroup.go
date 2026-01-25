package model

import (
	"time"
)

// PositionGroup（ポジションの正規化グループ）
type PositionGroup struct {
	PositionGroupID uint      `gorm:"primaryKey"`               // ポジショングループID
	Name            string    `gorm:"unique;size:100;not null"` // 正規化されたポジション名（例: "PM"）
	CreatedAt       time.Time // 作成日時
	UpdatedAt       time.Time // 更新日時

	Words []PositionWord `gorm:"foreignKey:PositionGroupID;references:PositionGroupID"` // 表記ゆれ一覧
}
