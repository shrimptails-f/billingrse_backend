package model

import "time"

// 登録単語1に対してKeywordGroupを複数登録するための中間テーブル
type KeywordGroupWordLink struct {
	KeywordGroupID uint `gorm:"primaryKey"`
	KeyWordID      uint `gorm:"primaryKey"`
	CreatedAt      time.Time

	// 循環してて完全に積んでるのでコメントアウト
	// KeyWord KeyWord `gorm:"foreignKey:KeyWordID;references:ID"`
}
