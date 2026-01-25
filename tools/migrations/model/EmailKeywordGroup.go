package model

import (
	"time"
)

// EmailKeywordGroup（メールとキーワードの多対多）
type EmailKeywordGroup struct {
	UserID         uint `gorm:"primaryKey"` // ユーザーID
	EmailID        uint `gorm:"primaryKey"` // メールID
	KeywordGroupID uint `gorm:"primaryKey"` // キーワードグループID
	CreatedAt      time.Time
	User           User `gorm:"foreignKey:UserID;references:ID;constraint:OnDelete:CASCADE"` // ユーザー

	// 循環してて完全に積んでるのでコメントアウト
	// KeywordGroup KeywordGroup `gorm:"foreignKey:KeywordGroupID;references:KeywordGroupID"`
}
